// Package conditionalrelease defines types for rule-based module gating (plan 1.11).
package conditionalrelease

import (
	"time"

	"github.com/google/uuid"
)

// CompletionMode is how a module's items must be completed.
type CompletionMode string

const (
	CompletionAllItems        CompletionMode = "all_items"
	CompletionOneItem         CompletionMode = "one_item"
	CompletionSequentialOrder CompletionMode = "sequential_order"
)

// RuleType is a per-item completion requirement.
type RuleType string

const (
	RuleMustView          RuleType = "must_view"
	RuleMustMarkDone      RuleType = "must_mark_done"
	RuleMustSubmit        RuleType = "must_submit"
	RuleMustScoreAtLeast  RuleType = "must_score_at_least"
	RuleMustContribute    RuleType = "must_contribute"
)

// ModuleRequirement is the instructor-authored module gating config.
type ModuleRequirement struct {
	ModuleID        uuid.UUID      `json:"moduleId"`
	CompletionMode  CompletionMode `json:"completionMode"`
	UnlockAt        *time.Time     `json:"unlockAt,omitempty"`
	PrerequisiteIDs []uuid.UUID    `json:"prerequisiteModuleIds,omitempty"`
}

// ItemRule is a per-item completion rule.
type ItemRule struct {
	ItemID    uuid.UUID `json:"itemId"`
	RuleType  RuleType  `json:"ruleType"`
	Threshold *float64  `json:"threshold,omitempty"`
}

// ItemProgress is one student_item_progress row.
type ItemProgress struct {
	ItemID       uuid.UUID
	Status       string
	MetAt        *time.Time
	EvidenceJSON []byte
}

// ModuleProgress is one student_module_progress row.
type ModuleProgress struct {
	ModuleID    uuid.UUID
	Status      string
	UnlockedAt  *time.Time
	CompletedAt *time.Time
}

// LockReason explains why content is locked.
type LockReason struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	ItemID  string `json:"itemId,omitempty"`
	Title   string `json:"title,omitempty"`
}

// ItemLockState is the gating state for one structure item.
type ItemLockState struct {
	ItemID   string     `json:"itemId"`
	Locked   bool       `json:"locked"`
	Complete bool       `json:"complete"`
	Reason   *LockReason `json:"reason,omitempty"`
}

// ModuleLockState is the gating state for one module.
type ModuleLockState struct {
	ModuleID  string          `json:"moduleId"`
	Title     string          `json:"title"`
	SortOrder int             `json:"sortOrder"`
	Locked    bool            `json:"locked"`
	Complete  bool            `json:"complete"`
	Reason    *LockReason     `json:"reason,omitempty"`
	Items     []ItemLockState `json:"items,omitempty"`
}

// StudentProgressSnapshot is the full lock/progress state for one enrollment.
type StudentProgressSnapshot struct {
	EnrollmentID string            `json:"enrollmentId"`
	Modules      []ModuleLockState `json:"modules"`
}

// ReportRow is one cell in the instructor requirements report.
type ReportRow struct {
	EnrollmentID string `json:"enrollmentId"`
	UserID       string `json:"userId"`
	DisplayName  string `json:"displayName"`
	Email        string `json:"email"`
	ItemID       string `json:"itemId"`
	ItemTitle    string `json:"itemTitle"`
	ModuleTitle  string `json:"moduleTitle"`
	RuleType     string `json:"ruleType,omitempty"`
	Status       string `json:"status"`
	MetAt        string `json:"metAt,omitempty"`
}
