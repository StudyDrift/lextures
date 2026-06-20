package coursestructure

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lextures/lextures/server/internal/repos/assignmentoverrides"
)

// ApplyAssignToForStudent resolves plan 2.15 assign-to targeting for one student's enrollment
// over a structure listing: it patches each assignment/quiz item's effective due date and
// drops items the student isn't targeted by (FR-3/FR-4). Non-assignment/quiz items pass through
// untouched. This is the single resolver used by the student dashboard, my-grades, and (via the
// calendar service) the calendar feed, replacing the section-only override application.
func ApplyAssignToForStudent(ctx context.Context, pool *pgxpool.Pool, enrollmentID uuid.UUID, items []ItemResponse) ([]ItemResponse, error) {
	var ids []uuid.UUID
	bases := make(map[uuid.UUID]assignmentoverrides.BaseDates)
	for i := range items {
		if items[i].Kind != "assignment" && items[i].Kind != "quiz" {
			continue
		}
		id, err := uuid.Parse(items[i].ID)
		if err != nil {
			continue
		}
		ids = append(ids, id)
		bases[id] = assignmentoverrides.BaseDates{DueAt: items[i].DueAt}
	}
	if len(ids) == 0 {
		return items, nil
	}
	effMap, err := assignmentoverrides.EffectiveForStudentBatch(ctx, pool, enrollmentID, ids, bases)
	if err != nil {
		return nil, err
	}
	out := make([]ItemResponse, 0, len(items))
	for i := range items {
		if items[i].Kind != "assignment" && items[i].Kind != "quiz" {
			out = append(out, items[i])
			continue
		}
		id, err := uuid.Parse(items[i].ID)
		if err != nil {
			out = append(out, items[i])
			continue
		}
		eff, ok := effMap[id]
		if !ok || !eff.Visible {
			continue
		}
		items[i].DueAt = eff.DueAt
		out = append(out, items[i])
	}
	return out, nil
}
