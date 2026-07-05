package httpserver

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/lextures/lextures/server/internal/apierr"
	"github.com/lextures/lextures/server/internal/repos/course"
	"github.com/lextures/lextures/server/internal/repos/courseevaluations"
	"github.com/lextures/lextures/server/internal/repos/enrollment"
)

func (d Deps) evaluationsFeatureOff(w http.ResponseWriter, r *http.Request) bool {
	if !d.effectiveConfig().FFCourseEvaluations {
		apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Course evaluations are not enabled.")
		return true
	}
	return false
}

// handleGetCourseEvaluationStatus returns whether an evaluation window is open and
// whether the current user has already submitted — no identity linkable to responses.
func (d Deps) handleGetCourseEvaluationStatus() http.HandlerFunc {
	type questionResp struct {
		Type     string   `json:"type"`
		Text     string   `json:"text"`
		Options  []string `json:"options,omitempty"`
		Required bool     `json:"required,omitempty"`
	}
	type resp struct {
		WindowOpen   bool           `json:"windowOpen"`
		WindowID     string         `json:"windowId,omitempty"`
		HasSubmitted bool           `json:"hasSubmitted"`
		OpensAt      string         `json:"opensAt,omitempty"`
		ClosesAt     string         `json:"closesAt,omitempty"`
		Questions    []questionResp `json:"questions,omitempty"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		if d.evaluationsFeatureOff(w, r) {
			return
		}
		courseCode, viewer, ok := d.requireCourseAccess(w, r)
		if !ok {
			return
		}
		cid, err := course.GetIDByCourseCode(r.Context(), d.Pool, courseCode)
		if err != nil || cid == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Course not found.")
			return
		}

		win, err := courseevaluations.GetActiveWindowByCourseID(r.Context(), d.Pool, *cid, time.Now().UTC())
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load evaluation status.")
			return
		}

		out := resp{}
		if win != nil {
			out.WindowOpen = true
			out.WindowID = win.ID.String()
			out.OpensAt = win.OpensAt.UTC().Format(time.RFC3339Nano)
			out.ClosesAt = win.ClosesAt.UTC().Format(time.RFC3339Nano)

			submitted, err := courseevaluations.HasUserSubmitted(r.Context(), d.Pool, win.ID, viewer)
			if err != nil {
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to check submission status.")
				return
			}
			out.HasSubmitted = submitted

			if !submitted {
				tmpl, err := courseevaluations.GetTemplate(r.Context(), d.Pool, win.TemplateID)
				if err == nil {
					_ = json.Unmarshal(tmpl.Questions, &out.Questions)
				}
			}
		}

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(out)
	}
}

// handlePostCourseEvaluationSubmit accepts an anonymous evaluation submission.
// The student's user ID is stored in evaluation_submissions but NOT in evaluation_responses.
func (d Deps) handlePostCourseEvaluationSubmit() http.HandlerFunc {
	type body struct {
		Answers json.RawMessage `json:"answers"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		if d.evaluationsFeatureOff(w, r) {
			return
		}
		courseCode, viewer, ok := d.requireCourseAccess(w, r)
		if !ok {
			return
		}
		_ = courseCode

		windowIDStr := chi.URLParam(r, "window_id")
		windowID, err := uuid.Parse(windowIDStr)
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid window ID.")
			return
		}

		// Verify student role — only students may submit.
		isStudent, err := enrollment.UserHasStudentEquivalentEnrollment(r.Context(), d.Pool, courseCode, viewer)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to verify enrollment.")
			return
		}
		if !isStudent {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "Only enrolled students may submit evaluations.")
			return
		}

		raw, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Failed to read request.")
			return
		}
		var b body
		if err := json.Unmarshal(raw, &b); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid request body.")
			return
		}

		in := courseevaluations.SubmitInput{
			WindowID: windowID,
			UserID:   viewer,
			Answers:  b.Answers,
		}
		err = courseevaluations.SubmitResponse(r.Context(), d.Pool, in, time.Now().UTC())
		if errors.Is(err, courseevaluations.ErrAlreadySubmitted) {
			apierr.WriteJSON(w, http.StatusConflict, apierr.CodeInvalidInput, "You have already submitted this evaluation.")
			return
		}
		if errors.Is(err, courseevaluations.ErrWindowClosed) {
			apierr.WriteJSON(w, http.StatusGone, apierr.CodeInvalidInput, "The evaluation window is closed.")
			return
		}
		if errors.Is(err, courseevaluations.ErrWindowNotOpen) {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "The evaluation window is not yet open.")
			return
		}
		if errors.Is(err, courseevaluations.ErrWindowNotFound) {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Evaluation window not found.")
			return
		}
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to submit evaluation.")
			return
		}

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]string{"message": "Thank you, your response has been recorded."})
	}
}

// handleGetCourseEvaluationResults returns aggregate results for the most recent closed window.
// Instructors see aggregate stats only; individual responses are never exposed.
func (d Deps) handleGetCourseEvaluationResults() http.HandlerFunc {
	type questionResultResp struct {
		Index        int            `json:"index"`
		Type         string         `json:"type"`
		Text         string         `json:"text"`
		Average      *float64       `json:"average,omitempty"`
		Distribution map[string]int `json:"distribution,omitempty"`
		OpenTexts    []string       `json:"openTexts,omitempty"`
	}
	type resp struct {
		WindowID       string               `json:"windowId"`
		OpensAt        string               `json:"opensAt"`
		ClosesAt       string               `json:"closesAt"`
		ResponseCount  int                  `json:"responseCount"`
		EnrolledCount  int                  `json:"enrolledCount"`
		CompletionPct  float64              `json:"completionPct"`
		MeetsThreshold bool                 `json:"meetsThreshold"`
		Questions      []questionResultResp `json:"questions"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		if d.evaluationsFeatureOff(w, r) {
			return
		}
		courseCode, viewer, ok := d.requireCourseAccess(w, r)
		if !ok {
			return
		}

		// Require instructor or admin role for results.
		isStaff, err := enrollment.UserIsCourseStaff(r.Context(), d.Pool, courseCode, viewer)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to verify permissions.")
			return
		}
		if !isStaff {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "Only course staff may view evaluation results.")
			return
		}

		cid, err := course.GetIDByCourseCode(r.Context(), d.Pool, courseCode)
		if err != nil || cid == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Course not found.")
			return
		}

		// Find most recent closed window.
		windows, err := courseevaluations.ListWindowsByCourseID(r.Context(), d.Pool, *cid)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load evaluation windows.")
			return
		}
		now := time.Now().UTC()
		var win *courseevaluations.Window
		for i := range windows {
			if windows[i].ClosesAt.Before(now) {
				win = &windows[i]
				break
			}
		}
		if win == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "No closed evaluation window found.")
			return
		}

		// Load template questions.
		tmpl, err := courseevaluations.GetTemplate(r.Context(), d.Pool, win.TemplateID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load evaluation template.")
			return
		}

		agg, err := courseevaluations.GetAggregateResults(r.Context(), d.Pool, win, tmpl.Questions)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to compute results.")
			return
		}

		var completionPct float64
		if win.EnrolledCount > 0 {
			completionPct = float64(win.ResponseCount) / float64(win.EnrolledCount) * 100
		}

		out := resp{
			WindowID:       win.ID.String(),
			OpensAt:        win.OpensAt.UTC().Format(time.RFC3339Nano),
			ClosesAt:       win.ClosesAt.UTC().Format(time.RFC3339Nano),
			ResponseCount:  win.ResponseCount,
			EnrolledCount:  win.EnrolledCount,
			CompletionPct:  completionPct,
			MeetsThreshold: agg.MeetsThreshold,
		}

		for _, q := range agg.Questions {
			qr := questionResultResp{
				Index:        q.QuestionIndex,
				Type:         q.Type,
				Text:         q.Text,
				Average:      q.Average,
				Distribution: q.Distribution,
			}
			if q.Type == courseevaluations.QuestionTypeOpenText {
				qr.OpenTexts = q.OpenTexts
			}
			out.Questions = append(out.Questions, qr)
		}

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(out)
	}
}

// handleAdminPostEvaluationWindow creates an evaluation window for a course (admin).
func (d Deps) handleAdminPostEvaluationWindow() http.HandlerFunc {
	type body struct {
		TemplateID string `json:"templateId"`
		OpensAt    string `json:"opensAt"`
		ClosesAt   string `json:"closesAt"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		if d.evaluationsFeatureOff(w, r) {
			return
		}
		_, ok := d.adminRbacUser(w, r)
		if !ok {
			return
		}

		courseCode := chi.URLParam(r, "course_code")
		if courseCode == "" {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Missing course code.")
			return
		}

		cid, err := course.GetIDByCourseCode(r.Context(), d.Pool, courseCode)
		if err != nil || cid == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Course not found.")
			return
		}

		raw, err := io.ReadAll(io.LimitReader(r.Body, 64*1024))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Failed to read request.")
			return
		}
		var b body
		if err := json.Unmarshal(raw, &b); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid request body.")
			return
		}

		tmplID, err := uuid.Parse(b.TemplateID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid template ID.")
			return
		}
		opensAt, err := time.Parse(time.RFC3339, b.OpensAt)
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid opensAt format; use RFC3339.")
			return
		}
		closesAt, err := time.Parse(time.RFC3339, b.ClosesAt)
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid closesAt format; use RFC3339.")
			return
		}
		if !closesAt.After(opensAt) {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "closesAt must be after opensAt.")
			return
		}

		// Count enrolled students for the window snapshot.
		roster, err := enrollment.ListRosterForCourse(r.Context(), d.Pool, courseCode)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to count enrollments.")
			return
		}
		enrolledCount := 0
		for _, row := range roster {
			if row.Role == "student" {
				enrolledCount++
			}
		}

		win, err := courseevaluations.CreateWindow(r.Context(), d.Pool, courseevaluations.CreateWindowInput{
			CourseID:      *cid,
			TemplateID:    tmplID,
			OpensAt:       opensAt.UTC(),
			ClosesAt:      closesAt.UTC(),
			EnrolledCount: enrolledCount,
		})
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to create evaluation window.")
			return
		}

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(windowToJSON(win))
	}
}


func windowToJSON(w *courseevaluations.Window) map[string]any {
	return map[string]any{
		"id":            w.ID.String(),
		"courseId":      w.CourseID.String(),
		"templateId":    w.TemplateID.String(),
		"opensAt":       w.OpensAt.UTC().Format(time.RFC3339Nano),
		"closesAt":      w.ClosesAt.UTC().Format(time.RFC3339Nano),
		"enrolledCount": w.EnrolledCount,
		"responseCount": w.ResponseCount,
		"createdAt":     w.CreatedAt.UTC().Format(time.RFC3339Nano),
	}
}
