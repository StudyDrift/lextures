// Package coursechecklist persists checklist dismissals, evaluation snapshots, and audit events (CC.2).
package coursechecklist

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// ItemState is one row from course.course_checklist_item_state.
type ItemState struct {
	CourseID          uuid.UUID
	ItemID            string
	DismissedAt       *time.Time
	DismissedByUserID *uuid.UUID
	DismissReason     string
	DismissNote       string
	SnoozedUntil      *time.Time
	RestoredAt        *time.Time
	RestoredByUserID  *uuid.UUID
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

// Snapshot is one row from course.course_checklist_snapshots.
type Snapshot struct {
	CourseID             uuid.UUID
	ComputedAt           time.Time
	EngineVersion        int
	CatalogVersion       string
	Payload              json.RawMessage
	TotalCount           int
	DoneCount            int
	OutstandingEssential int
	OutstandingTotal     int
	DismissedCount       int
}

// Event is one audit row from course.course_checklist_events.
type Event struct {
	ID          uuid.UUID
	CourseID    uuid.UUID
	ItemID      string
	Action      string
	ActorUserID *uuid.UUID
	Reason      string
	OccurredAt  time.Time
}

// DismissNoteExport is a DSAR export row for staff-authored dismiss notes.
type DismissNoteExport struct {
	CourseID     uuid.UUID
	CourseCode   string
	ItemID       string
	Reason       string
	Note         string
	DismissedAt  time.Time
}

// MutationFreshness is the latest course/structure mutation time used for staleness.
type MutationFreshness struct {
	CourseUpdatedAt time.Time
	StructureMaxAt  *time.Time
}

// LatestMutation returns the newest mutation timestamp among course and structure.
func (m MutationFreshness) LatestMutation() time.Time {
	latest := m.CourseUpdatedAt
	if m.StructureMaxAt != nil && m.StructureMaxAt.After(latest) {
		latest = *m.StructureMaxAt
	}
	return latest
}
