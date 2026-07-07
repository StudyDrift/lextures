package httpserver

import (
	"context"
	"encoding/json"
	"math"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/lextures/lextures/server/internal/apierr"
	"github.com/lextures/lextures/server/internal/courseroles"
	"github.com/lextures/lextures/server/internal/gradingredaction"
	"github.com/lextures/lextures/server/internal/repos/course"
	"github.com/lextures/lextures/server/internal/repos/coursegrades"
	"github.com/lextures/lextures/server/internal/repos/coursemoduleassignments"
	"github.com/lextures/lextures/server/internal/repos/enrollment"
	"github.com/lextures/lextures/server/internal/repos/moduleassignmentsubmissions"
	"github.com/lextures/lextures/server/internal/repos/provisionalgrades"
	"github.com/lextures/lextures/server/internal/repos/user"
	"github.com/lextures/lextures/server/internal/service/moderatedgrading"
)

func (d Deps) moderatedGradingFeatureOff(w http.ResponseWriter) bool {
	if !d.effectiveConfig().ModeratedGradingEnabled {
		apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Moderated grading is not enabled.")
		return true
	}
	return false
}

type moderationAssignmentContext struct {
	courseID   uuid.UUID
	courseCode string
	itemID     uuid.UUID
	assignRow  *coursemoduleassignments.CourseItemAssignmentRow
	viewer     uuid.UUID
	isModerator bool
}

func (d Deps) loadModerationAssignment(
	w http.ResponseWriter,
	r *http.Request,
	requireModerator bool,
) (*moderationAssignmentContext, bool) {
	if d.moderatedGradingFeatureOff(w) {
		return nil, false
	}
	courseCode, viewer, ok := d.requireCourseAccess(w, r)
	if !ok {
		return nil, false
	}
	itemID, err := uuid.Parse(chi.URLParam(r, "item_id"))
	if err != nil {
		apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid assignment id.")
		return nil, false
	}
	cid, err := course.GetIDByCourseCode(r.Context(), d.Pool, courseCode)
	if err != nil || cid == nil {
		apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Course not found.")
		return nil, false
	}
	assignRow, err := coursemoduleassignments.GetForCourseItem(r.Context(), d.Pool, *cid, itemID)
	if err != nil {
		apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load assignment.")
		return nil, false
	}
	if assignRow == nil || !assignRow.ModeratedGrading {
		apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Moderated grading is not enabled for this assignment.")
		return nil, false
	}
	canView, err := courseroles.UserHasPermission(r.Context(), d.Pool, viewer, "course:"+courseCode+":gradebook:view")
	if err != nil || !canView {
		apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "Gradebook permission required.")
		return nil, false
	}
	isModerator := assignRow.ModeratorUserID != nil && *assignRow.ModeratorUserID == viewer
	if requireModerator && !isModerator {
		apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "Only the assignment moderator may perform this action.")
		return nil, false
	}
	return &moderationAssignmentContext{
		courseID:    *cid,
		courseCode:  courseCode,
		itemID:      itemID,
		assignRow:   assignRow,
		viewer:      viewer,
		isModerator: isModerator,
	}, true
}

func (d Deps) submissionDisplayLabelsForModeration(
	ctx context.Context,
	courseCode string,
	assignRow *coursemoduleassignments.CourseItemAssignmentRow,
	submissions []moduleassignmentsubmissions.SubmissionRow,
) (map[uuid.UUID]string, map[uuid.UUID]string, error) {
	submissionLabels := make(map[uuid.UUID]string, len(submissions))
	studentLabels := make(map[uuid.UUID]string, len(submissions))
	if len(submissions) == 0 {
		return submissionLabels, studentLabels, nil
	}
	students, err := enrollment.ListStudentUsersForCourseCode(ctx, d.Pool, courseCode, nil)
	if err != nil {
		return nil, nil, err
	}
	roster := buildAssignmentRosterEntries(students, submissions)
	cfg := d.effectiveConfig()
	redact := gradingredaction.ShouldRedactSubmissionPiiForStaff(
		cfg.BlindGradingEnabled,
		assignRow.BlindGrading,
		assignRow.IdentitiesRevealedAt != nil,
	)
	displayNames := map[uuid.UUID]string{}
	if !redact {
		userIDs := make([]uuid.UUID, 0, len(roster))
		for _, entry := range roster {
			userIDs = append(userIDs, entry.UserID)
		}
		displayNames, err = user.DisplayLabelsByIDs(ctx, d.Pool, userIDs)
		if err != nil {
			return nil, nil, err
		}
		for _, entry := range roster {
			if strings.TrimSpace(displayNames[entry.UserID]) == "" && strings.TrimSpace(entry.DisplayName) != "" {
				displayNames[entry.UserID] = strings.TrimSpace(entry.DisplayName)
			}
		}
	}
	blindRanks := blindRanksForRoster(roster)
	for _, sub := range submissions {
		var label string
		if redact {
			label = gradingredaction.BlindStudentLabel(blindRanks[sub.SubmittedBy])
		} else {
			label = strings.TrimSpace(displayNames[sub.SubmittedBy])
			if label == "" {
				label = "Unknown student"
			}
		}
		studentLabels[sub.SubmittedBy] = label
		submissionLabels[sub.ID] = label
	}
	return submissionLabels, studentLabels, nil
}

func pointsWorthInt32(pw *int) *int32 {
	if pw == nil || *pw <= 0 {
		return nil
	}
	v := int32(*pw)
	return &v
}

// handleGetModerationReconciliation is GET .../assignments/{item_id}/reconciliation.
func (d Deps) handleGetModerationReconciliation() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		mctx, ok := d.loadModerationAssignment(w, r, true)
		if !ok {
			return
		}
		ctx := r.Context()
		submissions, err := moduleassignmentsubmissions.ListForAssignment(
			ctx, d.Pool, mctx.courseID, mctx.itemID, moduleassignmentsubmissions.GradedFilterAll,
		)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load submissions.")
			return
		}
		provRows, err := provisionalgrades.ListForAssignment(ctx, d.Pool, mctx.courseID, mctx.itemID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load provisional grades.")
			return
		}
		subByID := make(map[uuid.UUID]moduleassignmentsubmissions.SubmissionRow, len(submissions))
		for _, sub := range submissions {
			subByID[sub.ID] = sub
		}
		submissionLabels, studentLabels, err := d.submissionDisplayLabelsForModeration(
			ctx, mctx.courseCode, mctx.assignRow, submissions,
		)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to resolve submission labels.")
			return
		}
		graderIDs := make([]uuid.UUID, 0, len(provRows))
		seenGraders := map[uuid.UUID]struct{}{}
		for _, pg := range provRows {
			if _, ok := seenGraders[pg.GraderID]; ok {
				continue
			}
			seenGraders[pg.GraderID] = struct{}{}
			graderIDs = append(graderIDs, pg.GraderID)
		}
		graderNames, err := user.DisplayLabelsByIDs(ctx, d.Pool, graderIDs)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to resolve grader names.")
			return
		}
		bySubmission := make(map[uuid.UUID][]provisionalgrades.ProvisionalGradeRow)
		for _, pg := range provRows {
			bySubmission[pg.SubmissionID] = append(bySubmission[pg.SubmissionID], pg)
		}
		type gradeCell struct {
			points                *float64
			reconciliationSource  *string
		}
		finalByStudent := make(map[uuid.UUID]gradeCell)
		gradeRows, err := d.Pool.Query(ctx, `
SELECT student_user_id, points_earned, reconciliation_source
FROM course.course_grades
WHERE course_id = $1 AND module_item_id = $2
`, mctx.courseID, mctx.itemID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load final grades.")
			return
		}
		defer gradeRows.Close()
		for gradeRows.Next() {
			var studentID uuid.UUID
			var pts *float64
			var source *string
			if err := gradeRows.Scan(&studentID, &pts, &source); err != nil {
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to scan final grades.")
				return
			}
			finalByStudent[studentID] = gradeCell{points: pts, reconciliationSource: source}
		}
		threshold := int32(mctx.assignRow.ModerationThresholdPct)
		pw := pointsWorthInt32(mctx.assignRow.PointsWorth)
		rowsOut := make([]map[string]any, 0, len(bySubmission))
		unreconciledFlagged := 0
		submissionIDs := make([]uuid.UUID, 0, len(bySubmission))
		for sid := range bySubmission {
			submissionIDs = append(submissionIDs, sid)
		}
		sort.Slice(submissionIDs, func(i, j int) bool {
			return submissionIDs[i].String() < submissionIDs[j].String()
		})
		for _, sid := range submissionIDs {
			pgs := bySubmission[sid]
			sub, ok := subByID[sid]
			if !ok {
				continue
			}
			minScore, maxScore := pgs[0].Score, pgs[0].Score
			provJSON := make([]map[string]any, 0, len(pgs))
			for _, pg := range pgs {
				if pg.Score < minScore {
					minScore = pg.Score
				}
				if pg.Score > maxScore {
					maxScore = pg.Score
				}
				gname := strings.TrimSpace(graderNames[pg.GraderID])
				if gname == "" {
					gname = "Unknown grader"
				}
				var submittedAt *string
				if pg.SubmittedAt != nil {
					s := pg.SubmittedAt.UTC().Format(time.RFC3339)
					submittedAt = &s
				}
				provJSON = append(provJSON, map[string]any{
					"submissionId": sid.String(),
					"graderId":     pg.GraderID.String(),
					"graderName":   gname,
					"score":        pg.Score,
					"submittedAt":  submittedAt,
				})
			}
			flagged := moderatedgrading.ProvisionalScoresExceedThreshold(minScore, maxScore, pw, threshold)
			final := finalByStudent[sub.SubmittedBy]
			var finalScore any
			if final.points != nil {
				finalScore = *final.points
			}
			var reconSource any
			if final.reconciliationSource != nil {
				reconSource = *final.reconciliationSource
			}
			if flagged && final.reconciliationSource == nil {
				unreconciledFlagged++
			}
			studentName := studentLabels[sub.SubmittedBy]
			if studentName == "" {
				studentName = "Unknown student"
			}
			submissionLabel := submissionLabels[sid]
			if submissionLabel == "" {
				submissionLabel = studentName
			}
			var pointsWorth any
			if mctx.assignRow.PointsWorth != nil {
				pointsWorth = *mctx.assignRow.PointsWorth
			}
			rowsOut = append(rowsOut, map[string]any{
				"submissionId":         sid.String(),
				"studentUserId":        sub.SubmittedBy.String(),
				"studentName":          studentName,
				"submissionLabel":      submissionLabel,
				"provisional":          provJSON,
				"flagged":              flagged,
				"pointsWorth":          pointsWorth,
				"finalScore":           finalScore,
				"reconciliationSource": reconSource,
			})
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"rows":                     rowsOut,
			"unreconciledFlaggedCount": unreconciledFlagged,
		})
	}
}

type moderationReconcileBody struct {
	Action        string   `json:"action"`
	GraderID      *string  `json:"graderId"`
	OverrideScore *float64 `json:"overrideScore"`
}

// handlePostModerationReconcile is POST .../submissions/{submission_id}/reconcile.
func (d Deps) handlePostModerationReconcile() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		mctx, ok := d.loadModerationAssignment(w, r, true)
		if !ok {
			return
		}
		submissionID, err := uuid.Parse(chi.URLParam(r, "submission_id"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid submission id.")
			return
		}
		sub, err := moduleassignmentsubmissions.GetByIDForCourse(r.Context(), d.Pool, mctx.courseID, submissionID)
		if err != nil || sub == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Submission not found.")
			return
		}
		var body moderationReconcileBody
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		pgs, err := provisionalgrades.ListForAssignment(r.Context(), d.Pool, mctx.courseID, mctx.itemID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load provisional grades.")
			return
		}
		var forSub []provisionalgrades.ProvisionalGradeRow
		for _, pg := range pgs {
			if pg.SubmissionID == submissionID {
				forSub = append(forSub, pg)
			}
		}
		if len(forSub) == 0 {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "No provisional grades for this submission.")
			return
		}
		var finalScore float64
		var source string
		var reconciledGraderID *uuid.UUID
		switch body.Action {
		case "accept_grader":
			if body.GraderID == nil {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "graderId is required.")
				return
			}
			gid, err := uuid.Parse(*body.GraderID)
			if err != nil {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid graderId.")
				return
			}
			found := false
			for _, pg := range forSub {
				if pg.GraderID == gid {
					finalScore = pg.Score
					found = true
					reconciledGraderID = &gid
					break
				}
			}
			if !found {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Grader has no provisional score for this submission.")
				return
			}
			source = "grader"
		case "average":
			sum := 0.0
			for _, pg := range forSub {
				sum += pg.Score
			}
			finalScore = sum / float64(len(forSub))
			source = "average"
		case "single":
			if len(forSub) != 1 {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Single reconciliation requires exactly one provisional grade.")
				return
			}
			finalScore = forSub[0].Score
			gid := forSub[0].GraderID
			reconciledGraderID = &gid
			source = "single"
		case "override":
			if body.OverrideScore == nil || math.IsNaN(*body.OverrideScore) || math.IsInf(*body.OverrideScore, 0) {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "overrideScore is required.")
				return
			}
			finalScore = *body.OverrideScore
			source = "override"
		default:
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid action.")
			return
		}
		posting := mctx.assignRow.PostingPolicy
		if posting == "" {
			posting = "automatic"
		}
		if err := coursegrades.UpsertCell(r.Context(), d.Pool, mctx.courseID, sub.SubmittedBy, mctx.itemID, finalScore, nil, nil, posting); err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to save reconciled grade.")
			return
		}
		_, err = d.Pool.Exec(r.Context(), `
UPDATE course.course_grades
SET reconciliation_source = $1,
    reconciled_grader_id = $2,
    reconciled_by = $3,
    reconciled_at = NOW(),
    updated_at = NOW()
WHERE course_id = $4 AND student_user_id = $5 AND module_item_id = $6
`, source, reconciledGraderID, mctx.viewer, mctx.courseID, sub.SubmittedBy, mctx.itemID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to record reconciliation.")
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

// handleListProvisionalGrades is GET .../assignments/{item_id}/provisional-grades.
func (d Deps) handleListProvisionalGrades() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		mctx, ok := d.loadModerationAssignment(w, r, false)
		if !ok {
			return
		}
		var rows []provisionalgrades.ProvisionalGradeRow
		var err error
		if mctx.isModerator {
			rows, err = provisionalgrades.ListForAssignment(r.Context(), d.Pool, mctx.courseID, mctx.itemID)
		} else {
			rows, err = provisionalgrades.ListForAssignmentByGrader(r.Context(), d.Pool, mctx.courseID, mctx.itemID, mctx.viewer)
		}
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load provisional grades.")
			return
		}
		out := make([]map[string]any, 0, len(rows))
		for _, pg := range rows {
			var submittedAt *string
			if pg.SubmittedAt != nil {
				s := pg.SubmittedAt.UTC().Format(time.RFC3339)
				submittedAt = &s
			}
			out = append(out, map[string]any{
				"submissionId": pg.SubmissionID.String(),
				"graderId":     pg.GraderID.String(),
				"score":        pg.Score,
				"submittedAt":  submittedAt,
			})
		}
		writeJSON(w, http.StatusOK, map[string]any{"provisionalGrades": out})
	}
}

type provisionalGradePostBody struct {
	Score float64 `json:"score"`
}

// handlePostProvisionalGrade is POST .../submissions/{submission_id}/provisional-grades.
func (d Deps) handlePostProvisionalGrade() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		mctx, ok := d.loadModerationAssignment(w, r, false)
		if !ok {
			return
		}
		submissionID, err := uuid.Parse(chi.URLParam(r, "submission_id"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid submission id.")
			return
		}
		sub, err := moduleassignmentsubmissions.GetByIDForCourse(r.Context(), d.Pool, mctx.courseID, submissionID)
		if err != nil || sub == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Submission not found.")
			return
		}
		allowed := false
		for _, gid := range mctx.assignRow.ProvisionalGraderUserIDs {
			if gid == mctx.viewer {
				allowed = true
				break
			}
		}
		if !allowed && !mctx.isModerator {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "You are not assigned as a provisional grader.")
			return
		}
		var body provisionalGradePostBody
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		if math.IsNaN(body.Score) || math.IsInf(body.Score, 0) || body.Score < 0 {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid score.")
			return
		}
		if err := provisionalgrades.Upsert(r.Context(), d.Pool, submissionID, mctx.viewer, body.Score); err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to save provisional grade.")
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}