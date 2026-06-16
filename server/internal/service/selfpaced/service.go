// Package selfpaced implements self-paced enrollment progress, module gating, and
// completion logic for instructor-free courses (plan 15.2).
package selfpaced

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/lextures/lextures/server/internal/repos/learnerprogress"
)

// ProgressPercent returns the integer completion percentage (0–100). It returns 0 when
// there are no items so an empty course never reports 100% complete.
func ProgressPercent(completed, total int) int {
	if total <= 0 || completed <= 0 {
		return 0
	}
	if completed >= total {
		return 100
	}
	return int(float64(completed) / float64(total) * 100)
}

// IsCourseComplete reports whether every item in a non-empty course is completed.
func IsCourseComplete(completed, total int) bool {
	return total > 0 && completed >= total
}

// moduleComplete reports whether a module has all its leaf items completed. A module with
// no items is treated as complete so it never blocks a gated sequence.
func moduleComplete(m learnerprogress.ModuleProgress) bool {
	return m.TotalItems == 0 || m.CompletedItems >= m.TotalItems
}

// ModuleView is the per-module progress and gating state surfaced to the learner UI.
type ModuleView struct {
	ModuleID        string `json:"moduleId"`
	Title           string `json:"title"`
	SortOrder       int    `json:"sortOrder"`
	TotalItems      int    `json:"totalItems"`
	CompletedItems  int    `json:"completedItems"`
	ProgressPercent int    `json:"progressPercent"`
	Completed       bool   `json:"completed"`
	Locked          bool   `json:"locked"`
}

// Summary is the full self-paced progress snapshot for one enrollment.
type Summary struct {
	TotalItems        int          `json:"totalItems"`
	CompletedItems    int          `json:"completedItems"`
	ProgressPercent   int          `json:"progressPercent"`
	Completed         bool         `json:"completed"`
	GatingEnabled     bool         `json:"gatingEnabled"`
	Modules           []ModuleView `json:"modules"`
	LastVisitedItemID *string      `json:"lastVisitedItemId,omitempty"`
}

// BuildModuleViews computes per-module progress and gating. When gating is enabled a module
// is locked if any earlier module (by sort order) is incomplete. The input must be ordered
// by sort order.
func BuildModuleViews(modules []learnerprogress.ModuleProgress, gatingEnabled bool) []ModuleView {
	views := make([]ModuleView, 0, len(modules))
	priorComplete := true
	for _, m := range modules {
		complete := moduleComplete(m)
		view := ModuleView{
			ModuleID:        m.ModuleID.String(),
			Title:           m.Title,
			SortOrder:       m.SortOrder,
			TotalItems:      m.TotalItems,
			CompletedItems:  m.CompletedItems,
			ProgressPercent: ProgressPercent(m.CompletedItems, m.TotalItems),
			Completed:       complete,
			Locked:          gatingEnabled && !priorComplete,
		}
		views = append(views, view)
		// A locked module does not advance the gate; later modules stay locked too.
		priorComplete = priorComplete && complete
	}
	return views
}

// ModuleIsLocked reports whether the module containing itemID is locked for this enrollment
// under module gating. Returns false when gating is disabled.
func ModuleIsLocked(modules []learnerprogress.ModuleProgress, gatingEnabled bool, moduleID uuid.UUID) bool {
	if !gatingEnabled {
		return false
	}
	priorComplete := true
	for _, m := range modules {
		if m.ModuleID == moduleID {
			return !priorComplete
		}
		priorComplete = priorComplete && moduleComplete(m)
	}
	return false
}

// LoadSummary assembles the full progress snapshot for an enrollment from the database.
func LoadSummary(ctx context.Context, pool *pgxpool.Pool, courseID, enrollmentID uuid.UUID, gatingEnabled bool) (Summary, error) {
	totals, err := learnerprogress.CourseProgress(ctx, pool, courseID, enrollmentID)
	if err != nil {
		return Summary{}, err
	}
	modules, err := learnerprogress.ModuleProgressForEnrollment(ctx, pool, courseID, enrollmentID)
	if err != nil {
		return Summary{}, err
	}
	last, err := learnerprogress.LastVisitedItem(ctx, pool, enrollmentID)
	if err != nil {
		return Summary{}, err
	}
	s := Summary{
		TotalItems:      totals.TotalItems,
		CompletedItems:  totals.CompletedItems,
		ProgressPercent: ProgressPercent(totals.CompletedItems, totals.TotalItems),
		Completed:       IsCourseComplete(totals.CompletedItems, totals.TotalItems),
		GatingEnabled:   gatingEnabled,
		Modules:         BuildModuleViews(modules, gatingEnabled),
	}
	if last != nil {
		ls := last.String()
		s.LastVisitedItemID = &ls
	}
	return s, nil
}
