// Package gradeexport computes final letter grades and serialises them to CSV (plan 14.5).
package gradeexport

import (
	"bytes"
	"context"
	"encoding/csv"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lextures/lextures/server/internal/gradingdisplay"
	"github.com/lextures/lextures/server/internal/gradingdrops"
	"github.com/lextures/lextures/server/internal/repos/coursegrades"
	"github.com/lextures/lextures/server/internal/repos/coursegrading"
	"github.com/lextures/lextures/server/internal/repos/coursemoduleassignments"
	"github.com/lextures/lextures/server/internal/repos/coursestructure"
	"github.com/lextures/lextures/server/internal/repos/gradingschemes"
)

// StudentGrade is the computed final grade record for one student.
type StudentGrade struct {
	EnrollmentID   uuid.UUID
	UserID         uuid.UUID
	DisplayName    string
	ExternalSISID  string // SRN / external student ID (empty when not set)
	State          string // active, withdrawn, audit, incomplete, …
	ComputedGrade  string // letter grade derived from weighted gradebook
	FinalGrade     string // may be overridden by instructor
	OverrideReason string
}

// enrollmentRow is the raw enrollment row for grade computation.
type enrollmentRow struct {
	enrollmentID  uuid.UUID
	userID        uuid.UUID
	displayName   string
	externalSISID string
	state         string
}

func listStudentsForExport(ctx context.Context, pool *pgxpool.Pool, courseCode string) ([]enrollmentRow, error) {
	rows, err := pool.Query(ctx, `
SELECT ce.id, ce.user_id,
       COALESCE(NULLIF(TRIM(u.display_name), ''), u.email) AS display_label,
       COALESCE(ce.external_sis_id, '')                    AS external_sis_id,
       ce.state::text
FROM course.course_enrollments ce
INNER JOIN course.courses c ON c.id = ce.course_id
INNER JOIN "user".users u ON u.id = ce.user_id
WHERE c.course_code = $1
  AND ce.role = 'student'
  AND (
    (ce.state = 'active' AND ce.active)
    OR ce.state IN ('withdrawn', 'audit', 'no_credit', 'incomplete')
  )
ORDER BY display_label ASC, ce.user_id ASC
`, courseCode)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []enrollmentRow
	for rows.Next() {
		var r enrollmentRow
		if err := rows.Scan(&r.enrollmentID, &r.userID, &r.displayName, &r.externalSISID, &r.state); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// ComputeForCourse returns final grades for all active/former students in a course.
func ComputeForCourse(ctx context.Context, pool *pgxpool.Pool, courseID uuid.UUID, courseCode string) ([]StudentGrade, error) {
	students, err := listStudentsForExport(ctx, pool, courseCode)
	if err != nil {
		return nil, fmt.Errorf("list students: %w", err)
	}

	items, err := coursestructure.ListForCourseWithEnrichment(ctx, pool, courseID, true)
	if err != nil {
		return nil, fmt.Errorf("list course structure: %w", err)
	}

	dropFlags, err := coursemoduleassignments.ItemDropFlagsForCourse(ctx, pool, courseID)
	if err != nil {
		return nil, fmt.Errorf("load drop flags: %w", err)
	}

	groups, err := coursegrading.ListAssignmentGroups(ctx, pool, courseID)
	if err != nil {
		return nil, fmt.Errorf("load assignment groups: %w", err)
	}
	gpol := gradingdrops.GroupPoliciesFromSettings(groups)

	grades, _, _, excused, err := coursegrades.ListForCourse(ctx, pool, courseID)
	if err != nil {
		return nil, fmt.Errorf("load grades: %w", err)
	}

	schemeRow, err := gradingschemes.GetActiveForCourse(ctx, pool, courseID)
	if err != nil {
		return nil, fmt.Errorf("load grading scheme: %w", err)
	}

	var courseKind *gradingdisplay.Kind
	var parsed *gradingdisplay.ParsedScale
	if schemeRow != nil {
		k, ok := gradingdisplay.ParseKind(schemeRow.GradingDisplayType)
		if !ok {
			k = gradingdisplay.Points
		}
		courseKind = &k
		ps, err := gradingdisplay.ParseScale(k, schemeRow.ScaleJSON)
		if err == nil {
			parsed = &ps
		}
	}
	if parsed == nil {
		p := gradingdisplay.ParsedScale{Kind: gradingdisplay.Points}
		parsed = &p
	}

	// Build column metadata for drop-rule computation.
	var colMetaSlice []gradingdrops.ColMeta
	for i := range items {
		if items[i].Kind != "assignment" && items[i].Kind != "quiz" {
			continue
		}
		id, err := uuid.Parse(items[i].ID)
		if err != nil {
			continue
		}
		if items[i].PointsWorth == nil {
			continue
		}
		max := float64(*items[i].PointsWorth)
		if max <= 0 {
			continue
		}
		var gptr *uuid.UUID
		if items[i].AssignmentGroupID != nil {
			if gu, e := uuid.Parse(*items[i].AssignmentGroupID); e == nil {
				gptr = &gu
			}
		}
		df := dropFlags[id]
		colMetaSlice = append(colMetaSlice, gradingdrops.ColMeta{
			ID: id, GroupID: gptr, Max: max,
			NeverDrop: df.NeverDrop, ReplaceWithFinal: df.ReplaceWithFinal,
		})
	}

	out := make([]StudentGrade, 0, len(students))
	for _, s := range students {
		var computedGrade string
		switch s.state {
		case "withdrawn":
			computedGrade = "W"
		case "audit":
			computedGrade = "AU"
		case "incomplete":
			computedGrade = "I"
		case "no_credit":
			computedGrade = "NC"
		default:
			computedGrade = computeFinalGrade(s.userID.String(), grades, excused, colMetaSlice, gpol, parsed, courseKind)
		}
		out = append(out, StudentGrade{
			EnrollmentID:  s.enrollmentID,
			UserID:        s.userID,
			DisplayName:   s.displayName,
			ExternalSISID: s.externalSISID,
			State:         s.state,
			ComputedGrade: computedGrade,
			FinalGrade:    computedGrade,
		})
	}
	return out, nil
}

func computeFinalGrade(
	userID string,
	grades map[string]map[string]string,
	excused map[string]map[string]bool,
	cols []gradingdrops.ColMeta,
	gpol map[uuid.UUID]gradingdrops.GroupDropPolicy,
	parsed *gradingdisplay.ParsedScale,
	_ *gradingdisplay.Kind,
) string {
	earned := make(map[uuid.UUID]float64)
	if g, ok := grades[userID]; ok {
		for iid, ptsStr := range g {
			id, err := uuid.Parse(iid)
			if err != nil {
				continue
			}
			earned[id] = parsePoints(ptsStr)
		}
	}
	exc := make(map[uuid.UUID]bool)
	if e, ok := excused[userID]; ok {
		for iid, b := range e {
			id, err := uuid.Parse(iid)
			if err != nil {
				continue
			}
			exc[id] = b
		}
	}

	dropped := gradingdrops.ItemDropsForLearner(gpol, cols, earned, exc)
	var totalEarned, totalPossible float64
	for _, c := range cols {
		if dropped[c.ID] || exc[c.ID] {
			continue
		}
		totalPossible += c.Max
		if pts, ok := earned[c.ID]; ok && pts > 0 {
			totalEarned += pts
		}
	}
	if totalPossible == 0 {
		return ""
	}
	pct := totalEarned / totalPossible * 100
	return gradingdisplay.ToDisplayGrade(pct, nil, parsed, gradingdisplay.Percentage)
}

func parsePoints(s string) float64 {
	if s == "" {
		return 0
	}
	var f float64
	_, _ = fmt.Sscanf(s, "%f", &f)
	return f
}

// GenerateCSV serialises a grade list to Banner/Workday/Colleague-compatible CSV.
func GenerateCSV(grades []StudentGrade) ([]byte, error) {
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	if err := w.Write([]string{"StudentID", "StudentName", "FinalGrade", "EnrollmentState"}); err != nil {
		return nil, err
	}
	for _, g := range grades {
		sid := g.ExternalSISID
		if sid == "" {
			sid = g.UserID.String()
		}
		if err := w.Write([]string{sid, g.DisplayName, g.FinalGrade, g.State}); err != nil {
			return nil, err
		}
	}
	w.Flush()
	return buf.Bytes(), w.Error()
}
