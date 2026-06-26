package httpserver

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/lextures/lextures/server/internal/apierr"
	"github.com/lextures/lextures/server/internal/canvassubmissionsyncqueue"
	"github.com/lextures/lextures/server/internal/repos/canvasimportjobs"
	"github.com/lextures/lextures/server/internal/repos/course"
	"github.com/lextures/lextures/server/internal/repos/quizattempts"
	"github.com/lextures/lextures/server/internal/repos/user"
)

// registerQuizGradeSyncRoutes wires the quiz attempt → Canvas grade push.
func (d Deps) registerQuizGradeSyncRoutes(r chi.Router) {
	r.Post("/api/v1/courses/{course_code}/quizzes/{item_id}/attempts/{attempt_id}/sync-canvas", d.handlePostQuizAttemptSyncCanvas())
}

// handlePostQuizAttemptSyncCanvas queues a background job that pushes a quiz attempt's gradebook
// score to the linked Canvas course. It mirrors the assignment submission sync (same job + WS UX).
func (d Deps) handlePostQuizAttemptSyncCanvas() http.HandlerFunc {
	type body struct {
		CanvasBaseURL string   `json:"canvasBaseUrl"`
		AccessToken   string   `json:"accessToken"`
		PointsEarned  *float64 `json:"pointsEarned"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost+","+http.MethodOptions)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		if d.CanvasSubmissionSyncQueue == nil || d.CanvasSubmissionSyncJobs == nil {
			apierr.WriteJSON(w, http.StatusServiceUnavailable, apierr.CodeInternal, "Server misconfiguration.")
			return
		}
		courseCode := chi.URLParam(r, "course_code")
		viewer, ok := d.requireQuizGrader(w, r)
		if !ok {
			return
		}
		itemID, err := uuid.Parse(chi.URLParam(r, "item_id"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid item id.")
			return
		}
		attemptID, err := uuid.Parse(chi.URLParam(r, "attempt_id"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid attempt id.")
			return
		}

		ctx := r.Context()
		cid, err := course.GetIDByCourseCode(ctx, d.Pool, courseCode)
		if err != nil || cid == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Course not found.")
			return
		}
		attempt, err := quizattempts.GetAttemptResult(ctx, d.Pool, attemptID)
		if err != nil || attempt == nil || attempt.StructureItemID != itemID || attempt.CourseID != *cid {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Attempt not found.")
			return
		}

		payload, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Could not read body.")
			return
		}
		var b body
		if len(payload) > 0 {
			if err := json.Unmarshal(payload, &b); err != nil {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
				return
			}
		}

		link, err := canvasimportjobs.LatestLinkedForCourse(ctx, d.Pool, courseCode)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load Canvas link.")
			return
		}
		if link == nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "This course was not imported from Canvas.")
			return
		}
		courseRow, err := course.GetPublicByCourseCode(ctx, d.Pool, courseCode)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load course.")
			return
		}
		if courseRow == nil || !courseRow.CanvasGradeSyncEnabled {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Canvas grade sync is not enabled for this course.")
			return
		}
		canvasBaseRaw := strings.TrimSpace(b.CanvasBaseURL)
		if canvasBaseRaw == "" {
			canvasBaseRaw = link.CanvasBaseURL
		}
		canvasBase, err := normalizeCanvasBaseURL(canvasBaseRaw, d.effectiveConfig().CanvasAllowedHostSuffixes)
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, err.Error())
			return
		}
		token := strings.TrimSpace(b.AccessToken)
		if token == "" {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Canvas access token is required.")
			return
		}

		jobID := d.CanvasSubmissionSyncJobs.Create(viewer)
		msg := canvassubmissionsyncqueue.QueueMessage{
			JobID:         jobID,
			UserID:        viewer,
			CourseCode:    courseCode,
			ItemKind:      "quiz",
			ItemID:        itemID,
			SubmissionID:  attemptID,
			CanvasBaseURL: canvasBase,
			AccessToken:   token,
			PointsEarned:  b.PointsEarned,
		}
		if err := d.CanvasSubmissionSyncQueue.Publish(ctx, msg); err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to enqueue Canvas sync.")
			return
		}

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusAccepted)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"jobId":   jobID.String(),
			"message": "Canvas sync queued. You can keep grading — we will notify you when it finishes.",
		})
	}
}

// executeQuizGradeSyncToCanvas pushes a quiz attempt's gradebook score to the linked Canvas course.
// Canvas quizzes are backed by an assignment, so the grade is posted to that assignment's submission.
func (d Deps) executeQuizGradeSyncToCanvas(ctx context.Context, in submissionSyncCanvasInput) (map[string]any, error) {
	if d.Pool == nil {
		return nil, errors.New("server misconfiguration")
	}
	cid, err := course.GetIDByCourseCode(ctx, d.Pool, in.CourseCode)
	if err != nil || cid == nil {
		return nil, errors.New("Course not found.")
	}
	attempt, err := quizattempts.GetAttemptResult(ctx, d.Pool, in.SubmissionID)
	if err != nil || attempt == nil || attempt.StructureItemID != in.ItemID || attempt.CourseID != *cid {
		return nil, errors.New("Attempt not found.")
	}

	link, err := canvasimportjobs.LatestLinkedForCourse(ctx, d.Pool, in.CourseCode)
	if err != nil {
		return nil, errors.New("Failed to load Canvas link.")
	}
	if link == nil {
		return nil, errors.New("This course was not imported from Canvas.")
	}
	canvasBase, err := normalizeCanvasBaseURL(in.CanvasBaseURL, d.effectiveConfig().CanvasAllowedHostSuffixes)
	if err != nil {
		return nil, err
	}
	token := strings.TrimSpace(in.AccessToken)
	if token == "" {
		return nil, errors.New("Canvas access token is required.")
	}
	canvasCourseID, err := strconv.ParseInt(strings.TrimSpace(link.CanvasCourseID), 10, 64)
	if err != nil || canvasCourseID <= 0 {
		return nil, errors.New("Canvas course id is invalid for this import.")
	}

	var itemTitle string
	var storedCanvasAssignID *int64
	if err := d.Pool.QueryRow(ctx, `
		SELECT title, canvas_assignment_id FROM course.course_structure_items
		WHERE id = $1 AND course_id = $2 AND kind = 'quiz' AND archived = false`,
		in.ItemID, *cid).Scan(&itemTitle, &storedCanvasAssignID); err != nil {
		return nil, errors.New("Quiz not found.")
	}

	student, err := user.FindByID(ctx, d.Pool, attempt.StudentUserID)
	if err != nil || student == nil {
		return nil, errors.New("Failed to load student.")
	}

	// Quizzes have no rubric in Lextures; resolve the gradebook points for this student/quiz.
	pushGrade, gradeErr := resolveLexturesGradeForCanvasPush(
		ctx, d.Pool, *cid, attempt.StudentUserID, in.ItemID, nil, in.PointsEarned, nil, nil,
	)
	if gradeErr != nil {
		return nil, gradeErr
	}

	client := canvasHTTPClient()
	// Prefer the Canvas assignment id captured at import time; fall back to title matching only for
	// items imported before ids were persisted (titles are not unique, so this is best-effort).
	var canvasAssignID int64
	if storedCanvasAssignID != nil && *storedCanvasAssignID > 0 {
		canvasAssignID = *storedCanvasAssignID
	} else {
		canvasAssignID, err = canvasFindAssignmentIDByTitle(ctx, client, canvasBase, token, canvasCourseID, itemTitle)
		if err != nil {
			return nil, err
		}
	}
	canvasUserID, err := canvasFindCanvasUserIDForEmail(ctx, client, canvasBase, token, canvasCourseID, student.Email)
	if err != nil {
		return nil, err
	}
	canvasAssign, err := canvasFetchAssignmentForGradePush(ctx, client, canvasBase, token, canvasCourseID, canvasAssignID)
	if err != nil {
		return nil, err
	}

	form := canvasBuildCanvasGradePushForm(pushGrade, nil, canvasAssign)
	if len(form) == 0 {
		return nil, errors.New("No grade data to push to Canvas.")
	}

	path := fmt.Sprintf("courses/%d/assignments/%d/submissions/%d", canvasCourseID, canvasAssignID, canvasUserID)
	if _, err := canvasPutForm(ctx, client, canvasBase, token, path, form); err != nil {
		return nil, err
	}

	out := map[string]any{
		"attemptId":      in.SubmissionID.String(),
		"pointsEarned":   pushGrade.points,
		"excused":        pushGrade.excused,
		"syncedToCanvas": true,
	}
	return out, nil
}
