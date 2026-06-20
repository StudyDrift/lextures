package httpserver

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lextures/lextures/server/internal/repos/assignmentoverrides"
	"github.com/lextures/lextures/server/internal/repos/coursestructure"
	"github.com/lextures/lextures/server/internal/repos/enrollment"
)

// studentAssignToContext is the set of ids needed to resolve plan 2.15 assign-to targeting for one
// student in one course: their student enrollment id, section id (if any), and group ids.
type studentAssignToContext struct {
	EnrollmentID *uuid.UUID
	SectionID    *uuid.UUID
	GroupIDs     []uuid.UUID
}

func loadStudentAssignToContext(ctx context.Context, pool *pgxpool.Pool, courseID, userID uuid.UUID) (studentAssignToContext, error) {
	var out studentAssignToContext
	eid, err := enrollment.GetStudentEnrollmentID(ctx, pool, courseID, userID)
	if err != nil {
		return out, err
	}
	out.EnrollmentID = eid
	sid, err := enrollment.GetStudentSectionID(ctx, pool, courseID, userID)
	if err != nil {
		return out, err
	}
	out.SectionID = sid
	if eid != nil {
		gids, err := assignmentoverrides.StudentGroupIDs(ctx, pool, *eid)
		if err != nil {
			return out, err
		}
		out.GroupIDs = gids
	}
	return out, nil
}

// applyAssignToOverridesToItems drops assignment/quiz items not visible to the student under assign-to
// targeting (plan 2.15 FR-3), and overrides DueAt for items where a more specific target applies
// (FR-4). Items of other kinds pass through unchanged. Batches the override lookup in one query.
func applyAssignToOverridesToItems(ctx context.Context, pool *pgxpool.Pool, items []coursestructure.ItemResponse, sctx studentAssignToContext) ([]coursestructure.ItemResponse, error) {
	if sctx.EnrollmentID == nil {
		return items, nil
	}
	itemIDs := make([]uuid.UUID, 0, len(items))
	for i := range items {
		if items[i].Kind != "assignment" && items[i].Kind != "quiz" {
			continue
		}
		if id, err := uuid.Parse(items[i].ID); err == nil {
			itemIDs = append(itemIDs, id)
		}
	}
	overridesByItem, err := assignmentoverrides.ListForItems(ctx, pool, itemIDs)
	if err != nil {
		return nil, err
	}
	out := make([]coursestructure.ItemResponse, 0, len(items))
	for i := range items {
		it := items[i]
		if it.Kind != "assignment" && it.Kind != "quiz" {
			out = append(out, it)
			continue
		}
		itemID, err := uuid.Parse(it.ID)
		if err != nil {
			out = append(out, it)
			continue
		}
		visible, eff := assignmentoverrides.ResolveFromRows(overridesByItem[itemID], *sctx.EnrollmentID, sctx.SectionID, sctx.GroupIDs)
		if !visible {
			continue
		}
		if eff.DueAt != nil {
			t := *eff.DueAt
			it.DueAt = &t
		}
		out = append(out, it)
	}
	return out, nil
}
