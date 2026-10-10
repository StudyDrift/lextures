// Package coursefinalgrade computes one learner's course grade with the same rules as the gradebook
// FINAL column and the My grades total (assignment-group weights, drop rules, quiz question-total
// fallback, excused and held items), so other surfaces such as student progress agree with them.
package coursefinalgrade

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lextures/lextures/server/internal/gradingdrops"
	"github.com/lextures/lextures/server/internal/repos/coursegrades"
	"github.com/lextures/lextures/server/internal/repos/coursegrading"
	"github.com/lextures/lextures/server/internal/repos/coursemoduleassignments"
	"github.com/lextures/lextures/server/internal/repos/coursestructure"
	"github.com/lextures/lextures/server/internal/repos/enrollment"
)

// PercentForStudent returns the learner's course grade as a 0–100 percentage, or nil when no graded
// (or past-due) work counts yet. It mirrors the My grades view for a student: student-visible
// structure with assign-to overrides applied, and manually-held (unposted) grades left out.
func PercentForStudent(ctx context.Context, pool *pgxpool.Pool, courseID, userID uuid.UUID, now time.Time) (*float64, error) {
	items, err := coursestructure.ListForCourseWithEnrichment(ctx, pool, courseID, false)
	if err != nil {
		return nil, err
	}
	if eid, err := enrollment.GetStudentEnrollmentID(ctx, pool, courseID, userID); err == nil && eid != nil {
		if filtered, err := coursestructure.ApplyAssignToForStudent(ctx, pool, *eid, items); err == nil {
			items = filtered
		}
	}

	var assignIDs []uuid.UUID
	for i := range items {
		if items[i].Kind == "assignment" {
			if id, e := uuid.Parse(items[i].ID); e == nil {
				assignIDs = append(assignIDs, id)
			}
		}
	}
	posting, err := coursemoduleassignments.PostingByItemID(ctx, pool, courseID, assignIDs)
	if err != nil {
		return nil, err
	}
	dropFlags, err := coursemoduleassignments.ItemDropFlagsForCourse(ctx, pool, courseID)
	if err != nil {
		return nil, err
	}
	groupRows, err := coursegrading.ListAssignmentGroups(ctx, pool, courseID)
	if err != nil {
		return nil, err
	}
	allGrades, _, postedAt, allExcused, err := coursegrades.ListForCourse(ctx, pool, courseID)
	if err != nil {
		return nil, err
	}

	uid := userID.String()
	grades := make(map[uuid.UUID]string)
	for iid, v := range allGrades[uid] {
		if id, e := uuid.Parse(iid); e == nil {
			grades[id] = v
		}
	}
	excused := make(map[uuid.UUID]bool)
	for iid, b := range allExcused[uid] {
		if id, e := uuid.Parse(iid); e == nil && b {
			excused[id] = true
		}
	}

	cols := make([]gradingdrops.FinalCol, 0, len(items))
	for i := range items {
		if items[i].Kind != "assignment" && items[i].Kind != "quiz" {
			continue
		}
		id, e := uuid.Parse(items[i].ID)
		if e != nil {
			continue
		}
		mp := coursestructure.GradebookMaxPoints(&items[i])
		if mp == nil || *mp <= 0 {
			continue
		}
		// Manually-held grades stay out of the learner's total until posted.
		if p, ok := posting[id]; ok && p.Policy == "manual" {
			if _, graded := grades[id]; graded && postedAt[uid][items[i].ID] == nil {
				continue
			}
		}
		col := gradingdrops.FinalCol{
			ID:               id,
			MaxPoints:        float64(*mp),
			DueAt:            items[i].DueAt,
			NeverDrop:        dropFlags[id].NeverDrop,
			ReplaceWithFinal: dropFlags[id].ReplaceWithFinal,
		}
		if items[i].AssignmentGroupID != nil {
			if gu, e := uuid.Parse(*items[i].AssignmentGroupID); e == nil {
				col.GroupID = &gu
			}
		}
		cols = append(cols, col)
	}

	policies := gradingdrops.GroupPoliciesFromSettings(groupRows)
	groups := make([]gradingdrops.FinalGroup, 0, len(groupRows))
	for _, g := range groupRows {
		groups = append(groups, gradingdrops.FinalGroup{ID: g.ID, WeightPercent: g.WeightPercent, Policy: policies[g.ID]})
	}
	return gradingdrops.CourseFinalPercent(cols, grades, groups, excused, now), nil
}
