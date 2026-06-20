// Package competencygating implements rule-based module requirements and conditional release (plan 1.11).
package competencygating

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/lextures/lextures/server/internal/models/conditionalrelease"
	crrepo "github.com/lextures/lextures/server/internal/repos/conditionalrelease"
)

// ErrSelfPrerequisite is returned when module_id == prerequisite_module_id.
var ErrSelfPrerequisite = errors.New("competencygating: a module cannot be a prerequisite of itself")

// ErrCircularPrerequisite is returned when an edge would close a cycle.
var ErrCircularPrerequisite = errors.New("competencygating: prerequisite would create a circular dependency")

// Service evaluates module requirements and student progress.
type Service struct {
	Pool *pgxpool.Pool
}

// New returns a Service wired to the given pool.
func New(pool *pgxpool.Pool) Service {
	return Service{Pool: pool}
}

// Health returns a stable service heartbeat string for wiring/tests.
func (s Service) Health(ctx context.Context) (string, error) {
	if ctx == nil {
		return "", fmt.Errorf("context is nil")
	}
	return "competencygating:ok", nil
}

// courseContext holds loaded gating data for one course/enrollment evaluation.
type courseContext struct {
	modules      []crrepo.CourseModule
	moduleReqs   map[uuid.UUID]conditionalrelease.ModuleRequirement
	itemRules    map[uuid.UUID]conditionalrelease.ItemRule
	itemProgress map[uuid.UUID]conditionalrelease.ItemProgress
	modProgress  map[uuid.UUID]conditionalrelease.ModuleProgress
	overrides    map[uuid.UUID]bool
	moduleItems  map[uuid.UUID][]crrepo.ModuleLeafItem
}

func (s Service) loadCourseContext(ctx context.Context, courseID, enrollmentID uuid.UUID) (*courseContext, error) {
	modules, err := crrepo.ListCourseModules(ctx, s.Pool, courseID)
	if err != nil {
		return nil, err
	}
	reqs, err := crrepo.ListModuleRequirementsForCourse(ctx, s.Pool, courseID)
	if err != nil {
		return nil, err
	}
	reqMap := make(map[uuid.UUID]conditionalrelease.ModuleRequirement, len(reqs))
	for _, r := range reqs {
		reqMap[r.ModuleID] = r
	}
	rules, err := crrepo.ListItemRulesForCourse(ctx, s.Pool, courseID)
	if err != nil {
		return nil, err
	}
	itemProgress, err := crrepo.ListItemProgressForEnrollment(ctx, s.Pool, enrollmentID)
	if err != nil {
		return nil, err
	}
	modProgress, err := crrepo.ListModuleProgressForEnrollment(ctx, s.Pool, enrollmentID)
	if err != nil {
		return nil, err
	}
	overrides, err := crrepo.ListUnlockOverridesForEnrollment(ctx, s.Pool, enrollmentID)
	if err != nil {
		return nil, err
	}
	moduleItems := make(map[uuid.UUID][]crrepo.ModuleLeafItem, len(modules))
	for _, m := range modules {
		items, err := crrepo.ListModuleLeafItems(ctx, s.Pool, m.ModuleID)
		if err != nil {
			return nil, err
		}
		moduleItems[m.ModuleID] = items
	}
	return &courseContext{
		modules:      modules,
		moduleReqs:   reqMap,
		itemRules:    rules,
		itemProgress: itemProgress,
		modProgress:  modProgress,
		overrides:    overrides,
		moduleItems:  moduleItems,
	}, nil
}

func (cc *courseContext) itemMet(itemID uuid.UUID) bool {
	if p, ok := cc.itemProgress[itemID]; ok && p.Status == "complete" {
		return true
	}
	return false
}

func (cc *courseContext) moduleComplete(moduleID uuid.UUID) bool {
	if p, ok := cc.modProgress[moduleID]; ok && p.Status == "complete" {
		return true
	}
	req, hasReq := cc.moduleReqs[moduleID]
	items := cc.moduleItems[moduleID]
	if !hasReq && len(items) == 0 {
		return true
	}
	mode := conditionalrelease.CompletionAllItems
	if hasReq {
		mode = req.CompletionMode
	}
	ruleItems := filterRuleItems(items, cc.itemRules)
	if len(ruleItems) == 0 && !hasReq {
		return allItemsMet(items, cc)
	}
	switch mode {
	case conditionalrelease.CompletionOneItem:
		for _, item := range ruleItems {
			if cc.itemMet(item.ItemID) {
				return true
			}
		}
		return len(ruleItems) == 0
	case conditionalrelease.CompletionSequentialOrder:
		for _, item := range ruleItems {
			if !cc.itemMet(item.ItemID) {
				return false
			}
		}
		return true
	default: // all_items
		for _, item := range ruleItems {
			if _, hasRule := cc.itemRules[item.ItemID]; hasRule && !cc.itemMet(item.ItemID) {
				return false
			}
		}
		return true
	}
}

func filterRuleItems(items []crrepo.ModuleLeafItem, rules map[uuid.UUID]conditionalrelease.ItemRule) []crrepo.ModuleLeafItem {
	var out []crrepo.ModuleLeafItem
	for _, item := range items {
		if _, ok := rules[item.ItemID]; ok {
			out = append(out, item)
		}
	}
	return out
}

func allItemsMet(items []crrepo.ModuleLeafItem, cc *courseContext) bool {
	for _, item := range items {
		if _, hasRule := cc.itemRules[item.ItemID]; hasRule && !cc.itemMet(item.ItemID) {
			return false
		}
	}
	return true
}

func (cc *courseContext) moduleUnlocked(moduleID uuid.UUID, now time.Time) (bool, *conditionalrelease.LockReason) {
	if cc.overrides[moduleID] {
		return true, nil
	}
	req, hasReq := cc.moduleReqs[moduleID]
	if hasReq && req.UnlockAt != nil && now.Before(req.UnlockAt.UTC()) {
		return false, &conditionalrelease.LockReason{
			Code:    "unlock_date",
			Message: fmt.Sprintf("This module unlocks on %s.", req.UnlockAt.UTC().Format(time.RFC3339)),
		}
	}
	if hasReq {
		for _, prereqID := range req.PrerequisiteIDs {
			if !cc.moduleComplete(prereqID) {
				title := prereqTitle(cc, prereqID)
				return false, &conditionalrelease.LockReason{
					Code:    "module_prerequisite",
					Message: fmt.Sprintf("Complete module %q to unlock.", title),
					Title:   title,
				}
			}
		}
	}
	return true, nil
}

func prereqTitle(cc *courseContext, moduleID uuid.UUID) string {
	for _, m := range cc.modules {
		if m.ModuleID == moduleID {
			return m.Title
		}
	}
	return "the previous module"
}

func (cc *courseContext) itemLocked(itemID uuid.UUID, now time.Time) (bool, *conditionalrelease.LockReason) {
	moduleID, errModule := findModuleForItem(cc, itemID)
	if errModule != nil {
		return false, nil
	}
	if moduleID == uuid.Nil {
		return false, nil
	}
	unlocked, modReason := cc.moduleUnlocked(moduleID, now)
	if !unlocked {
		return true, modReason
	}
	req, hasReq := cc.moduleReqs[moduleID]
	if hasReq && req.CompletionMode == conditionalrelease.CompletionSequentialOrder {
		items := cc.moduleItems[moduleID]
		for _, item := range items {
			if item.ItemID == itemID {
				break
			}
			if _, hasRule := cc.itemRules[item.ItemID]; hasRule && !cc.itemMet(item.ItemID) {
				return true, &conditionalrelease.LockReason{
					Code:    "sequential_order",
					Message: fmt.Sprintf("Complete %q first.", item.Title),
					ItemID:  item.ItemID.String(),
					Title:   item.Title,
				}
			}
		}
	}
	return false, nil
}

func findModuleForItem(cc *courseContext, itemID uuid.UUID) (uuid.UUID, error) {
	for moduleID, items := range cc.moduleItems {
		for _, item := range items {
			if item.ItemID == itemID {
				return moduleID, nil
			}
		}
	}
	return uuid.Nil, nil
}

// BuildStudentProgress computes lock/progress state for one enrollment.
func (s Service) BuildStudentProgress(ctx context.Context, courseID, enrollmentID uuid.UUID, now time.Time) (conditionalrelease.StudentProgressSnapshot, error) {
	cc, err := s.loadCourseContext(ctx, courseID, enrollmentID)
	if err != nil {
		return conditionalrelease.StudentProgressSnapshot{}, err
	}
	out := conditionalrelease.StudentProgressSnapshot{EnrollmentID: enrollmentID.String()}
	for _, m := range cc.modules {
		mod := conditionalrelease.ModuleLockState{
			ModuleID:  m.ModuleID.String(),
			Title:     m.Title,
			SortOrder: m.SortOrder,
		}
		unlocked, modReason := cc.moduleUnlocked(m.ModuleID, now)
		mod.Complete = cc.moduleComplete(m.ModuleID)
		mod.Locked = !unlocked
		mod.Reason = modReason
		for _, item := range cc.moduleItems[m.ModuleID] {
			itemState := conditionalrelease.ItemLockState{
				ItemID:   item.ItemID.String(),
				Complete: cc.itemMet(item.ItemID),
			}
			if !unlocked {
				itemState.Locked = true
				itemState.Reason = modReason
			} else {
				locked, reason := cc.itemLocked(item.ItemID, now)
				itemState.Locked = locked
				itemState.Reason = reason
			}
			mod.Items = append(mod.Items, itemState)
		}
		out.Modules = append(out.Modules, mod)
	}
	return out, nil
}

// CheckItemAccess reports whether a student may access an item. Instructors bypass gating.
func (s Service) CheckItemAccess(
	ctx context.Context, courseID, enrollmentID, userID, itemID uuid.UUID, now time.Time,
) (bool, *conditionalrelease.LockReason, error) {
	hasReq, err := crrepo.CourseHasRequirements(ctx, s.Pool, courseID)
	if err != nil {
		return false, nil, err
	}
	if !hasReq {
		return true, nil, nil
	}
	cc, err := s.loadCourseContext(ctx, courseID, enrollmentID)
	if err != nil {
		return false, nil, err
	}
	locked, reason := cc.itemLocked(itemID, now)
	return !locked, reason, nil
}

// EvaluateAndPersistItemRule checks whether an item's rule is met and updates progress idempotently.
func (s Service) EvaluateAndPersistItemRule(
	ctx context.Context, courseID, enrollmentID, userID, itemID uuid.UUID,
) (bool, error) {
	rule, err := crrepo.GetItemRule(ctx, s.Pool, itemID)
	if err != nil {
		return false, err
	}
	if rule == nil {
		return false, nil
	}
	met, evidence, err := s.evaluateRule(ctx, courseID, enrollmentID, userID, itemID, *rule)
	if err != nil {
		return false, err
	}
	if !met {
		return false, nil
	}
	newly, err := crrepo.MarkItemComplete(ctx, s.Pool, enrollmentID, itemID, evidence)
	if err != nil {
		return false, err
	}
	if newly {
		if err := s.recomputeModuleProgress(ctx, courseID, enrollmentID); err != nil {
			return newly, err
		}
	}
	return newly, nil
}

func (s Service) evaluateRule(
	ctx context.Context, courseID, enrollmentID, userID, itemID uuid.UUID, rule conditionalrelease.ItemRule,
) (bool, any, error) {
	switch rule.RuleType {
	case conditionalrelease.RuleMustView:
		ok, err := crrepo.ItemWasViewed(ctx, s.Pool, enrollmentID, itemID)
		return ok, map[string]string{"type": "view"}, err
	case conditionalrelease.RuleMustMarkDone:
		ok, err := crrepo.ItemWasMarkedDone(ctx, s.Pool, enrollmentID, itemID)
		return ok, map[string]string{"type": "mark_done"}, err
	case conditionalrelease.RuleMustSubmit:
		ok, err := crrepo.ItemHasSubmission(ctx, s.Pool, courseID, userID, itemID)
		return ok, map[string]string{"type": "submit"}, err
	case conditionalrelease.RuleMustScoreAtLeast:
		pct, err := crrepo.ItemScorePercent(ctx, s.Pool, courseID, userID, itemID)
		if err != nil || pct == nil || rule.Threshold == nil {
			return false, nil, err
		}
		met := *pct >= *rule.Threshold
		evidence := map[string]any{"type": "score", "percent": *pct, "threshold": *rule.Threshold}
		return met, evidence, nil
	case conditionalrelease.RuleMustContribute:
		ok, err := crrepo.ItemHasDiscussionContribution(ctx, s.Pool, courseID, userID, itemID)
		return ok, map[string]string{"type": "contribute"}, err
	default:
		var never = rule.RuleType
		_ = never
		return false, nil, fmt.Errorf("competencygating: unknown rule type %q", rule.RuleType)
	}
}

func (s Service) recomputeModuleProgress(ctx context.Context, courseID, enrollmentID uuid.UUID) error {
	cc, err := s.loadCourseContext(ctx, courseID, enrollmentID)
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	for _, m := range cc.modules {
		_, hasReq := cc.moduleReqs[m.ModuleID]
		if !hasReq && len(filterRuleItems(cc.moduleItems[m.ModuleID], cc.itemRules)) == 0 {
			continue
		}
		unlocked, _ := cc.moduleUnlocked(m.ModuleID, now)
		if cc.moduleComplete(m.ModuleID) {
			if err := crrepo.UpsertModuleProgress(ctx, s.Pool, enrollmentID, m.ModuleID, "complete"); err != nil {
				return err
			}
		} else if unlocked {
			if err := crrepo.UpsertModuleProgress(ctx, s.Pool, enrollmentID, m.ModuleID, "unlocked"); err != nil {
				return err
			}
		}
	}
	return nil
}

// SetModuleRequirements upserts module requirements and prerequisites with cycle detection.
func (s Service) SetModuleRequirements(
	ctx context.Context, moduleID uuid.UUID, mode conditionalrelease.CompletionMode, unlockAt *time.Time, prerequisiteIDs []uuid.UUID,
) error {
	for _, pid := range prerequisiteIDs {
		if pid == moduleID {
			return ErrSelfPrerequisite
		}
	}
	tx, err := s.Pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	for _, pid := range prerequisiteIDs {
		reaches, err := crrepo.PrerequisiteReachesModule(ctx, tx, pid, moduleID)
		if err != nil {
			return err
		}
		if reaches {
			return ErrCircularPrerequisite
		}
	}
	if err := crrepo.UpsertModuleRequirementTx(ctx, tx, moduleID, mode, unlockAt); err != nil {
		return err
	}
	if err := crrepo.SetModulePrerequisites(ctx, tx, moduleID, prerequisiteIDs); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// GrantUnlockOverride manually unlocks a module for one enrollment and records audit metadata in evidence.
func (s Service) GrantUnlockOverride(ctx context.Context, enrollmentID, moduleID, grantedBy uuid.UUID) error {
	if err := crrepo.InsertUnlockOverride(ctx, s.Pool, enrollmentID, moduleID, grantedBy); err != nil {
		return err
	}
	return crrepo.UpsertModuleProgress(ctx, s.Pool, enrollmentID, moduleID, "unlocked")
}
