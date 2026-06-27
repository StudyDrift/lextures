package httpserver

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/lextures/lextures/server/internal/apierr"
	"github.com/lextures/lextures/server/internal/courseroles"
	"github.com/lextures/lextures/server/internal/models/coursemodulequiz"
	"github.com/lextures/lextures/server/internal/repos/course"
	"github.com/lextures/lextures/server/internal/repos/coursemodulequizzes"
	"github.com/lextures/lextures/server/internal/repos/quizattempts"
	"github.com/lextures/lextures/server/internal/repos/rbac"
)

// handleQuizAttemptsList is GET /api/v1/courses/{course_code}/quizzes/{item_id}/attempts.
func (d Deps) handleQuizAttemptsList() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		courseCode, viewer, ok := d.requireCourseAccess(w, r)
		if !ok {
			return
		}
		itemID, err := uuid.Parse(chi.URLParam(r, "item_id"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid item id.")
			return
		}
		ctx := r.Context()
		cid, err := course.GetIDByCourseCode(ctx, d.Pool, courseCode)
		if err != nil || cid == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Course not found.")
			return
		}
		row, err := coursemodulequizzes.GetForCourseItem(ctx, d.Pool, *cid, itemID)
		if err != nil || row == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Not found.")
			return
		}

		canGradebook, err := courseroles.UserHasPermission(ctx, d.Pool, viewer, "course:"+courseCode+":gradebook:view")
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to verify permissions.")
			return
		}
		canConfigureAgent, cfgErr := rbac.UserHasPermission(ctx, d.Pool, viewer, "course:"+courseCode+":item:create")
		if cfgErr != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to verify permissions.")
			return
		}
		canViewAll := canGradebook || canConfigureAgent

		var filterStudent *uuid.UUID
		userIDParam := strings.TrimSpace(r.URL.Query().Get("userId"))
		if userIDParam != "" {
			parsed, perr := uuid.Parse(userIDParam)
			if perr != nil {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid user id.")
				return
			}
			if !canViewAll && parsed != viewer {
				apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "You do not have permission to view these attempts.")
				return
			}
			filterStudent = &parsed
		} else if !canViewAll {
			filterStudent = &viewer
		}

		attemptRows, err := quizattempts.ListSubmittedAttemptsForItem(ctx, d.Pool, *cid, itemID, filterStudent)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load quiz attempts.")
			return
		}

		attempts := make([]coursemodulequiz.QuizAttemptSummary, 0, len(attemptRows))
		for _, ar := range attemptRows {
			summary := coursemodulequiz.QuizAttemptSummary{
				ID:             ar.ID,
				AttemptNumber:  ar.AttemptNumber,
				SubmittedAt:    ar.SubmittedAt,
				ScorePercent:   ar.ScorePercent,
				PointsEarned:   ar.PointsEarned,
				PointsPossible: ar.PointsPossible,
			}
			if canViewAll {
				name := ar.StudentDisplayName
				summary.StudentName = &name
				sid := ar.StudentUserID
				summary.StudentUserID = &sid
				summary.NeedsManualGrading = ar.NeedsManualGrading
			}
			attempts = append(attempts, summary)
		}

		out := coursemodulequiz.QuizAttemptsListResponse{
			Attempts:     attempts,
			RetakePolicy: row.GradeAttemptPolicy,
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(out)
	}
}