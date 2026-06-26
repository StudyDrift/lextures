// Package conditionalrelease persists module requirements, item rules, and student progress (plan 1.11).
package conditionalrelease

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/lextures/lextures/server/internal/models/conditionalrelease"
)

// CourseHasRequirements reports whether a course has any module requirements or item rules.
func CourseHasRequirements(ctx context.Context, pool *pgxpool.Pool, courseID uuid.UUID) (bool, error) {
	var found bool
	err := pool.QueryRow(ctx, `
SELECT EXISTS (
    SELECT 1 FROM course.module_requirements mr
    INNER JOIN course.course_structure_items m ON m.id = mr.module_id
    WHERE m.course_id = $1 AND NOT m.archived
    UNION ALL
    SELECT 1 FROM course.item_completion_rules icr
    INNER JOIN course.course_structure_items i ON i.id = icr.item_id
    WHERE i.course_id = $1 AND NOT i.archived
    LIMIT 1
)
`, courseID).Scan(&found)
	return found, err
}

// UpsertModuleRequirementTx is the transactional variant of UpsertModuleRequirement.
func UpsertModuleRequirementTx(
	ctx context.Context, tx pgx.Tx, moduleID uuid.UUID, mode conditionalrelease.CompletionMode, unlockAt *time.Time,
) error {
	return upsertModuleRequirementExec(ctx, tx, moduleID, mode, unlockAt)
}

func upsertModuleRequirementExec(
	ctx context.Context, exec interface {
		Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
	},
	moduleID uuid.UUID, mode conditionalrelease.CompletionMode, unlockAt *time.Time,
) error {
	_, err := exec.Exec(ctx, `
INSERT INTO course.module_requirements (module_id, completion_mode, unlock_at, updated_at)
VALUES ($1, $2, $3, NOW())
ON CONFLICT (module_id) DO UPDATE SET
    completion_mode = EXCLUDED.completion_mode,
    unlock_at = EXCLUDED.unlock_at,
    updated_at = NOW()
`, moduleID, mode, unlockAt)
	return err
}

// GetModuleRequirement loads module requirement config including prerequisite IDs.
func GetModuleRequirement(ctx context.Context, pool *pgxpool.Pool, moduleID uuid.UUID) (*conditionalrelease.ModuleRequirement, error) {
	var req conditionalrelease.ModuleRequirement
	var unlockAt *time.Time
	err := pool.QueryRow(ctx, `
SELECT module_id, completion_mode, unlock_at
FROM course.module_requirements
WHERE module_id = $1
`, moduleID).Scan(&req.ModuleID, &req.CompletionMode, &unlockAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	req.UnlockAt = unlockAt
	rows, err := pool.Query(ctx, `
SELECT prerequisite_module_id FROM course.module_prerequisites WHERE module_id = $1
`, moduleID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var pid uuid.UUID
		if err := rows.Scan(&pid); err != nil {
			return nil, err
		}
		req.PrerequisiteIDs = append(req.PrerequisiteIDs, pid)
	}
	return &req, rows.Err()
}

// ListModuleRequirementsForCourse returns all module requirements in a course ordered by sort_order.
func ListModuleRequirementsForCourse(ctx context.Context, pool *pgxpool.Pool, courseID uuid.UUID) ([]conditionalrelease.ModuleRequirement, error) {
	rows, err := pool.Query(ctx, `
SELECT mr.module_id, mr.completion_mode, mr.unlock_at
FROM course.module_requirements mr
INNER JOIN course.course_structure_items m ON m.id = mr.module_id
WHERE m.course_id = $1 AND NOT m.archived
ORDER BY m.sort_order
`, courseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []conditionalrelease.ModuleRequirement
	for rows.Next() {
		var req conditionalrelease.ModuleRequirement
		var unlockAt *time.Time
		if err := rows.Scan(&req.ModuleID, &req.CompletionMode, &unlockAt); err != nil {
			return nil, err
		}
		req.UnlockAt = unlockAt
		out = append(out, req)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for i := range out {
		prereqs, err := listPrerequisites(ctx, pool, out[i].ModuleID)
		if err != nil {
			return nil, err
		}
		out[i].PrerequisiteIDs = prereqs
	}
	return out, nil
}

func listPrerequisites(ctx context.Context, pool *pgxpool.Pool, moduleID uuid.UUID) ([]uuid.UUID, error) {
	rows, err := pool.Query(ctx, `
SELECT prerequisite_module_id FROM course.module_prerequisites WHERE module_id = $1
`, moduleID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

// SetModulePrerequisites replaces all prerequisite edges for a module inside a transaction.
func SetModulePrerequisites(ctx context.Context, tx pgx.Tx, moduleID uuid.UUID, prerequisiteIDs []uuid.UUID) error {
	if _, err := tx.Exec(ctx, `DELETE FROM course.module_prerequisites WHERE module_id = $1`, moduleID); err != nil {
		return err
	}
	for _, pid := range prerequisiteIDs {
		if err := InsertPrerequisiteEdge(ctx, tx, moduleID, pid); err != nil {
			return err
		}
	}
	return nil
}

// UpsertItemRule sets or replaces the completion rule for an item.
func UpsertItemRule(ctx context.Context, pool *pgxpool.Pool, itemID uuid.UUID, ruleType conditionalrelease.RuleType, threshold *float64) error {
	_, err := pool.Exec(ctx, `
INSERT INTO course.item_completion_rules (item_id, rule_type, threshold, updated_at)
VALUES ($1, $2, $3, NOW())
ON CONFLICT (item_id) DO UPDATE SET
    rule_type = EXCLUDED.rule_type,
    threshold = EXCLUDED.threshold,
    updated_at = NOW()
`, itemID, ruleType, threshold)
	return err
}

// DeleteItemRule removes a completion rule from an item.
func DeleteItemRule(ctx context.Context, pool *pgxpool.Pool, itemID uuid.UUID) error {
	_, err := pool.Exec(ctx, `DELETE FROM course.item_completion_rules WHERE item_id = $1`, itemID)
	return err
}

// GetItemRule loads the rule for one item.
func GetItemRule(ctx context.Context, pool *pgxpool.Pool, itemID uuid.UUID) (*conditionalrelease.ItemRule, error) {
	var rule conditionalrelease.ItemRule
	var threshold *float64
	err := pool.QueryRow(ctx, `
SELECT item_id, rule_type, threshold FROM course.item_completion_rules WHERE item_id = $1
`, itemID).Scan(&rule.ItemID, &rule.RuleType, &threshold)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	rule.Threshold = threshold
	return &rule, nil
}

// ListItemRulesForCourse returns all item rules in a course keyed by item id.
func ListItemRulesForCourse(ctx context.Context, pool *pgxpool.Pool, courseID uuid.UUID) (map[uuid.UUID]conditionalrelease.ItemRule, error) {
	rows, err := pool.Query(ctx, `
SELECT icr.item_id, icr.rule_type, icr.threshold
FROM course.item_completion_rules icr
INNER JOIN course.course_structure_items i ON i.id = icr.item_id
WHERE i.course_id = $1 AND NOT i.archived
`, courseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make(map[uuid.UUID]conditionalrelease.ItemRule)
	for rows.Next() {
		var rule conditionalrelease.ItemRule
		var threshold *float64
		if err := rows.Scan(&rule.ItemID, &rule.RuleType, &threshold); err != nil {
			return nil, err
		}
		rule.Threshold = threshold
		out[rule.ItemID] = rule
	}
	return out, rows.Err()
}

// ModuleLeafItem is a published leaf item in a module ordered by sort_order.
type ModuleLeafItem struct {
	ItemID    uuid.UUID
	Title     string
	SortOrder int
	Kind      string
}

// ListModuleLeafItems returns published non-heading leaf items for a module.
func ListModuleLeafItems(ctx context.Context, pool *pgxpool.Pool, moduleID uuid.UUID) ([]ModuleLeafItem, error) {
	rows, err := pool.Query(ctx, `
SELECT id, title, sort_order, kind
FROM course.course_structure_items
WHERE parent_id = $1 AND kind NOT IN ('heading')
  AND published AND NOT archived
ORDER BY sort_order
`, moduleID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ModuleLeafItem
	for rows.Next() {
		var item ModuleLeafItem
		if err := rows.Scan(&item.ItemID, &item.Title, &item.SortOrder, &item.Kind); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

// CourseModule is a module row in sort order.
type CourseModule struct {
	ModuleID  uuid.UUID
	Title     string
	SortOrder int
}

// ListCourseModules returns non-archived modules for a course.
func ListCourseModules(ctx context.Context, pool *pgxpool.Pool, courseID uuid.UUID) ([]CourseModule, error) {
	rows, err := pool.Query(ctx, `
SELECT id, title, sort_order
FROM course.course_structure_items
WHERE course_id = $1 AND kind = 'module' AND NOT archived
ORDER BY sort_order
`, courseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []CourseModule
	for rows.Next() {
		var m CourseModule
		if err := rows.Scan(&m.ModuleID, &m.Title, &m.SortOrder); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

// MarkItemComplete idempotently marks an item requirement as complete for an enrollment.
// Returns true when the row newly transitioned to complete.
func MarkItemComplete(ctx context.Context, pool *pgxpool.Pool, enrollmentID, itemID uuid.UUID, evidence any) (bool, error) {
	var evidenceJSON []byte
	if evidence != nil {
		var err error
		evidenceJSON, err = json.Marshal(evidence)
		if err != nil {
			return false, err
		}
	}
	var alreadyComplete bool
	err := pool.QueryRow(ctx, `
WITH existing AS (
    SELECT status FROM course.student_item_progress
    WHERE enrollment_id = $1 AND item_id = $2
)
INSERT INTO course.student_item_progress (enrollment_id, item_id, status, met_at, evidence_json, updated_at)
VALUES ($1, $2, 'complete', NOW(), $3, NOW())
ON CONFLICT (enrollment_id, item_id) DO UPDATE SET
    status = 'complete',
    met_at = COALESCE(course.student_item_progress.met_at, NOW()),
    evidence_json = COALESCE($3, course.student_item_progress.evidence_json),
    updated_at = NOW()
RETURNING COALESCE((SELECT status = 'complete' FROM existing), false)
`, enrollmentID, itemID, evidenceJSON).Scan(&alreadyComplete)
	if err != nil {
		return false, err
	}
	return !alreadyComplete, nil
}

// ListItemProgressForEnrollment returns item progress keyed by item id.
func ListItemProgressForEnrollment(ctx context.Context, pool *pgxpool.Pool, enrollmentID uuid.UUID) (map[uuid.UUID]conditionalrelease.ItemProgress, error) {
	rows, err := pool.Query(ctx, `
SELECT item_id, status, met_at, evidence_json
FROM course.student_item_progress
WHERE enrollment_id = $1
`, enrollmentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make(map[uuid.UUID]conditionalrelease.ItemProgress)
	for rows.Next() {
		var p conditionalrelease.ItemProgress
		if err := rows.Scan(&p.ItemID, &p.Status, &p.MetAt, &p.EvidenceJSON); err != nil {
			return nil, err
		}
		out[p.ItemID] = p
	}
	return out, rows.Err()
}

// UpsertModuleProgress sets module progress status for an enrollment.
func UpsertModuleProgress(ctx context.Context, pool *pgxpool.Pool, enrollmentID, moduleID uuid.UUID, status string) error {
	var unlockedAt, completedAt *time.Time
	now := time.Now().UTC()
	switch status {
	case "unlocked":
		unlockedAt = &now
	case "complete":
		unlockedAt = &now
		completedAt = &now
	}
	_, err := pool.Exec(ctx, `
INSERT INTO course.student_module_progress (enrollment_id, module_id, status, unlocked_at, completed_at, updated_at)
VALUES ($1, $2, $3, $4, $5, NOW())
ON CONFLICT (enrollment_id, module_id) DO UPDATE SET
    status = EXCLUDED.status,
    unlocked_at = COALESCE(course.student_module_progress.unlocked_at, EXCLUDED.unlocked_at),
    completed_at = COALESCE(EXCLUDED.completed_at, course.student_module_progress.completed_at),
    updated_at = NOW()
`, enrollmentID, moduleID, status, unlockedAt, completedAt)
	return err
}

// ListModuleProgressForEnrollment returns module progress keyed by module id.
func ListModuleProgressForEnrollment(ctx context.Context, pool *pgxpool.Pool, enrollmentID uuid.UUID) (map[uuid.UUID]conditionalrelease.ModuleProgress, error) {
	rows, err := pool.Query(ctx, `
SELECT module_id, status, unlocked_at, completed_at
FROM course.student_module_progress
WHERE enrollment_id = $1
`, enrollmentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make(map[uuid.UUID]conditionalrelease.ModuleProgress)
	for rows.Next() {
		var p conditionalrelease.ModuleProgress
		if err := rows.Scan(&p.ModuleID, &p.Status, &p.UnlockedAt, &p.CompletedAt); err != nil {
			return nil, err
		}
		out[p.ModuleID] = p
	}
	return out, rows.Err()
}

// ListUnlockOverridesForEnrollment returns overridden module ids.
func ListUnlockOverridesForEnrollment(ctx context.Context, pool *pgxpool.Pool, enrollmentID uuid.UUID) (map[uuid.UUID]bool, error) {
	rows, err := pool.Query(ctx, `
SELECT module_id FROM course.module_unlock_overrides WHERE enrollment_id = $1
`, enrollmentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make(map[uuid.UUID]bool)
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out[id] = true
	}
	return out, rows.Err()
}

// InsertUnlockOverride grants a manual module unlock for one enrollment.
func InsertUnlockOverride(ctx context.Context, pool *pgxpool.Pool, enrollmentID, moduleID, grantedBy uuid.UUID) error {
	_, err := pool.Exec(ctx, `
INSERT INTO course.module_unlock_overrides (enrollment_id, module_id, granted_by)
VALUES ($1, $2, $3)
ON CONFLICT (enrollment_id, module_id) DO UPDATE SET
    granted_by = EXCLUDED.granted_by,
    granted_at = NOW()
`, enrollmentID, moduleID, grantedBy)
	return err
}
