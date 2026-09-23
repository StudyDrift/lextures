package httpserver

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/lextures/lextures/server/internal/apierr"
	"github.com/lextures/lextures/server/internal/repos/attendance"
	"github.com/lextures/lextures/server/internal/repos/coursemoduleassignments"
	"github.com/lextures/lextures/server/internal/repos/coursesections"
	gradingagentrepo "github.com/lextures/lextures/server/internal/repos/gradingagent"
	"github.com/lextures/lextures/server/internal/repos/groupspaces"
	"github.com/lextures/lextures/server/internal/repos/quizattempts"
)

// graderAgentFilterError is a caller-facing rejection of a run filter (403).
type graderAgentFilterError string

func (e graderAgentFilterError) Error() string { return string(e) }

// latestQuizAttempts keeps one submitted attempt per student: the highest attempt
// number, with a later row winning ties. Matches the grader canvas attempt list.
func latestQuizAttempts(rows []quizattempts.AttemptListRow) []quizattempts.AttemptListRow {
	if len(rows) == 0 {
		return nil
	}
	byStudent := make(map[uuid.UUID]quizattempts.AttemptListRow, len(rows))
	order := make([]uuid.UUID, 0, len(rows))
	for _, row := range rows {
		existing, ok := byStudent[row.StudentUserID]
		if !ok {
			order = append(order, row.StudentUserID)
			byStudent[row.StudentUserID] = row
			continue
		}
		if row.AttemptNumber >= existing.AttemptNumber {
			byStudent[row.StudentUserID] = row
		}
	}
	out := make([]quizattempts.AttemptListRow, 0, len(order))
	for _, studentID := range order {
		out = append(out, byStudent[studentID])
	}
	return out
}

// selectQuizAttemptTargets applies scope and optional student / attempt filters.
// allowedStudents nil means every student; onlyAttemptIDs nil means the latest
// attempt per student. A non-nil onlyAttemptIDs set selects those attempts exactly.
func selectQuizAttemptTargets(
	rows []quizattempts.AttemptListRow,
	scope gradingagentrepo.RunScope,
	submissionID string,
	overwrite bool,
	allowedStudents map[uuid.UUID]struct{},
	onlyAttemptIDs map[uuid.UUID]struct{},
) ([]quizattempts.AttemptListRow, error) {
	var pool []quizattempts.AttemptListRow
	if onlyAttemptIDs != nil {
		pool = make([]quizattempts.AttemptListRow, 0, len(onlyAttemptIDs))
		for _, row := range rows {
			if _, ok := onlyAttemptIDs[row.ID]; ok {
				pool = append(pool, row)
			}
		}
	} else {
		pool = latestQuizAttempts(rows)
	}
	if allowedStudents != nil {
		next := make([]quizattempts.AttemptListRow, 0, len(pool))
		for _, row := range pool {
			if _, ok := allowedStudents[row.StudentUserID]; ok {
				next = append(next, row)
			}
		}
		pool = next
	}
	switch scope {
	case gradingagentrepo.RunScopeCurrent:
		sid, err := uuid.Parse(strings.TrimSpace(submissionID))
		if err != nil {
			return nil, errInvalidScope("submissionId is required for current scope")
		}
		for _, row := range pool {
			if row.ID == sid {
				return []quizattempts.AttemptListRow{row}, nil
			}
		}
		return nil, errInvalidScope("submission not found")
	case gradingagentrepo.RunScopeUngraded:
		next := make([]quizattempts.AttemptListRow, 0, len(pool))
		for _, row := range pool {
			if row.NeedsManualGrading {
				next = append(next, row)
			}
		}
		return next, nil
	case gradingagentrepo.RunScopeAll:
		if !overwrite {
			return nil, errInvalidScope("overwrite confirmation required for all scope")
		}
		return pool, nil
	default:
		return nil, errInvalidScope("invalid scope")
	}
}

func userIDSet(ids []uuid.UUID) map[uuid.UUID]struct{} {
	out := make(map[uuid.UUID]struct{}, len(ids))
	for _, id := range ids {
		out[id] = struct{}{}
	}
	return out
}

func intersectUserSet(base map[uuid.UUID]struct{}, ids []uuid.UUID) map[uuid.UUID]struct{} {
	next := make(map[uuid.UUID]struct{}, len(ids))
	for _, id := range ids {
		if base == nil {
			next[id] = struct{}{}
			continue
		}
		if _, ok := base[id]; ok {
			next[id] = struct{}{}
		}
	}
	return next
}

func userIDsInSections(ctx context.Context, pool *pgxpool.Pool, sectionIDs []uuid.UUID) ([]uuid.UUID, error) {
	seen := make(map[uuid.UUID]struct{})
	out := make([]uuid.UUID, 0)
	for _, sectionID := range sectionIDs {
		roster, err := attendance.ListRosterForSection(ctx, pool, sectionID)
		if err != nil {
			return nil, err
		}
		for _, student := range roster {
			if _, ok := seen[student.UserID]; ok {
				continue
			}
			seen[student.UserID] = struct{}{}
			out = append(out, student.UserID)
		}
	}
	return out, nil
}

func listGroupMemberUserIDs(ctx context.Context, pool *pgxpool.Pool, groupID uuid.UUID) ([]uuid.UUID, error) {
	rows, err := pool.Query(ctx, `
SELECT e.user_id
FROM course.enrollment_group_memberships m
JOIN course.course_enrollments e ON e.id = m.enrollment_id
WHERE m.group_id = $1 AND e.active
`, groupID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]uuid.UUID, 0)
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

func labelsForQuizAttempts(rows []quizattempts.AttemptListRow, ids []uuid.UUID) map[uuid.UUID]string {
	byID := make(map[uuid.UUID]quizattempts.AttemptListRow, len(rows))
	for _, row := range rows {
		byID[row.ID] = row
	}
	out := make(map[uuid.UUID]string, len(ids))
	for _, id := range ids {
		row, ok := byID[id]
		if !ok {
			out[id] = id.String()[:8]
			continue
		}
		name := strings.TrimSpace(row.StudentDisplayName)
		if name == "" {
			name = row.StudentUserID.String()[:8]
		}
		out[id] = name
	}
	return out
}

func (d Deps) graderAgentReviewLabels(
	ctx context.Context,
	item *gradingAgentModuleItem,
	submissionIDs []uuid.UUID,
) (map[uuid.UUID]string, error) {
	if item != nil && item.Kind == "quiz" {
		rows, err := quizattempts.ListSubmittedAttemptsForItem(ctx, d.Pool, item.CourseID, item.ItemID, nil)
		if err != nil {
			return nil, err
		}
		return labelsForQuizAttempts(rows, submissionIDs), nil
	}
	var courseID, itemID uuid.UUID
	if item != nil {
		courseID = item.CourseID
		itemID = item.ItemID
	}
	assignRow, err := coursemoduleassignments.GetForCourseItem(ctx, d.Pool, courseID, itemID)
	if err != nil {
		return nil, err
	}
	return d.submissionLabelsForGraderAgentReview(ctx, courseID, itemID, assignRow, submissionIDs)
}

func (d Deps) resolveQuizGraderAgentTargets(
	ctx context.Context,
	courseCode string,
	courseID, itemID, viewer uuid.UUID,
	scope gradingagentrepo.RunScope,
	submissionID string,
	overwrite bool,
	runFilter *gradingagentrepo.RunFilter,
) ([]quizattempts.AttemptListRow, gradingagentrepo.RunScope, *graderAgentRunFilterContext, error) {
	attempts, err := quizattempts.ListSubmittedAttemptsForItem(ctx, d.Pool, courseID, itemID, nil)
	if err != nil {
		return nil, scope, nil, err
	}
	visible, err := d.graderAgentVisibleSectionIDs(ctx, courseID, courseCode, viewer)
	if err != nil {
		return nil, scope, nil, err
	}
	var allow map[uuid.UUID]struct{}
	if len(visible) > 0 {
		ids, idsErr := userIDsInSections(ctx, d.Pool, visible)
		if idsErr != nil {
			return nil, scope, nil, idsErr
		}
		allow = userIDSet(ids)
	}
	var meta *graderAgentRunFilterContext
	if runFilter != nil && runFilter.SectionID != nil {
		sec, secErr := coursesections.GetByID(ctx, d.Pool, courseID, *runFilter.SectionID)
		if secErr != nil {
			return nil, scope, nil, secErr
		}
		if sec == nil || sec.Status == "archived" {
			return nil, scope, nil, graderAgentFilterError("section not found")
		}
		if !sectionAllowed(visible, *runFilter.SectionID) {
			return nil, scope, nil, graderAgentFilterError("you do not have access to that section")
		}
		roster, rosterErr := attendance.ListRosterForSection(ctx, d.Pool, *runFilter.SectionID)
		if rosterErr != nil {
			return nil, scope, nil, rosterErr
		}
		ids := make([]uuid.UUID, 0, len(roster))
		for _, student := range roster {
			ids = append(ids, student.UserID)
		}
		allow = intersectUserSet(allow, ids)
		label := sec.SectionCode
		if sec.Name != nil && strings.TrimSpace(*sec.Name) != "" {
			label = strings.TrimSpace(*sec.Name)
		}
		if meta == nil {
			meta = &graderAgentRunFilterContext{}
		}
		meta.SectionLabel = &label
	}
	if runFilter != nil && runFilter.GroupID != nil {
		group, groupErr := groupspaces.GetGroupByCourseAndID(ctx, d.Pool, courseCode, *runFilter.GroupID)
		if groupErr != nil {
			return nil, scope, nil, groupErr
		}
		if group == nil {
			return nil, scope, nil, graderAgentFilterError("group not found")
		}
		members, memberErr := listGroupMemberUserIDs(ctx, d.Pool, *runFilter.GroupID)
		if memberErr != nil {
			return nil, scope, nil, memberErr
		}
		allow = intersectUserSet(allow, members)
		name := strings.TrimSpace(group.Name)
		if meta == nil {
			meta = &graderAgentRunFilterContext{}
		}
		meta.GroupLabel = &name
	}
	var only map[uuid.UUID]struct{}
	if runFilter != nil && len(runFilter.SubmissionIDs) > 0 {
		known := make(map[uuid.UUID]struct{}, len(attempts))
		for _, row := range attempts {
			known[row.ID] = struct{}{}
		}
		only = make(map[uuid.UUID]struct{}, len(runFilter.SubmissionIDs))
		for _, id := range runFilter.SubmissionIDs {
			if _, ok := known[id]; !ok {
				return nil, scope, nil, graderAgentFilterError("submission not found")
			}
			only[id] = struct{}{}
		}
	}
	selected, selErr := selectQuizAttemptTargets(attempts, scope, submissionID, overwrite, allow, only)
	if selErr != nil {
		return nil, scope, meta, selErr
	}
	return selected, scope, meta, nil
}

func writeGraderAgentTargetError(w http.ResponseWriter, err error) bool {
	if err == nil {
		return false
	}
	var scopeErr invalidScopeError
	if errors.As(err, &scopeErr) {
		apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, err.Error())
		return true
	}
	var denied graderAgentFilterError
	if errors.As(err, &denied) {
		apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, err.Error())
		return true
	}
	apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to resolve grading targets.")
	return true
}
