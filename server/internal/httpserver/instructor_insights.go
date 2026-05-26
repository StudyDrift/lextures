package httpserver

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/lextures/lextures/server/internal/apierr"
	"github.com/lextures/lextures/server/internal/courseroles"
	"github.com/lextures/lextures/server/internal/repos/course"
	"github.com/lextures/lextures/server/internal/service/instructorinsights"
)

func (d Deps) insightsFeatureEnabled(w http.ResponseWriter) bool {
	if !d.effectiveConfig().InstructorInsightsEnabled {
		apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Instructor insights is not enabled.")
		return false
	}
	return true
}

// requireInsightsInstructor authenticates, checks the feature flag, and asserts gradebook:view.
func (d Deps) requireInsightsInstructor(w http.ResponseWriter, r *http.Request) (courseCode string, viewer uuid.UUID, ok bool) {
	courseCode, viewer, ok = d.requireCourseAccess(w, r)
	if !ok {
		return "", uuid.UUID{}, false
	}
	if !d.insightsFeatureEnabled(w) {
		return "", uuid.UUID{}, false
	}
	has, err := courseroles.UserHasPermission(r.Context(), d.Pool, viewer, "course:"+courseCode+":gradebook:view")
	if err != nil {
		apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to verify permissions.")
		return "", uuid.UUID{}, false
	}
	if !has {
		apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "You do not have permission to view course insights.")
		return "", uuid.UUID{}, false
	}
	return courseCode, viewer, true
}

// handleGetCourseInsights is GET /api/v1/courses/{course_code}/analytics/insights
func (d Deps) handleGetCourseInsights() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		courseCode, _, ok := d.requireInsightsInstructor(w, r)
		if !ok {
			return
		}
		ctx := r.Context()
		cid, err := course.GetIDByCourseCode(ctx, d.Pool, courseCode)
		if err != nil || cid == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Course not found.")
			return
		}
		insights, err := instructorinsights.Load(ctx, d.Pool, *cid)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load insights.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(insights)
	}
}

// handleGetCrossSection is GET /api/v1/courses/{course_code}/analytics/cross-section
func (d Deps) handleGetCrossSection() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		courseCode, _, ok := d.requireInsightsInstructor(w, r)
		if !ok {
			return
		}
		ctx := r.Context()
		cid, err := course.GetIDByCourseCode(ctx, d.Pool, courseCode)
		if err != nil || cid == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Course not found.")
			return
		}
		rows, err := instructorinsights.LoadCrossSection(ctx, d.Pool, *cid)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load cross-section data.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(rows)
	}
}

type dismissSignalBody struct {
	SignalKey string `json:"signalKey"`
	Reason    string `json:"reason"`
}

// handleDismissInsightSignal is POST /api/v1/courses/{course_code}/analytics/insights/dismiss
func (d Deps) handleDismissInsightSignal() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		courseCode, viewer, ok := d.requireInsightsInstructor(w, r)
		if !ok {
			return
		}
		b, _ := io.ReadAll(r.Body)
		_ = r.Body.Close()
		var body dismissSignalBody
		if err := json.Unmarshal(b, &body); err != nil || body.SignalKey == "" {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "signalKey is required.")
			return
		}
		ctx := r.Context()
		cid, err := course.GetIDByCourseCode(ctx, d.Pool, courseCode)
		if err != nil || cid == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Course not found.")
			return
		}
		if err := instructorinsights.DismissSignal(ctx, d.Pool, *cid, viewer, body.SignalKey, body.Reason); err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to dismiss signal.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{"dismissed": true})
	}
}

// handleRefreshInsights is POST /api/v1/courses/{course_code}/analytics/insights/refresh
func (d Deps) handleRefreshInsights() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		courseCode, _, ok := d.requireInsightsInstructor(w, r)
		if !ok {
			return
		}
		ctx := r.Context()
		cid, err := course.GetIDByCourseCode(ctx, d.Pool, courseCode)
		if err != nil || cid == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Course not found.")
			return
		}
		insights, err := instructorinsights.Compute(ctx, d.Pool, *cid)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to compute insights.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(insights)
	}
}

func (d Deps) registerInsightsRoutes(r chi.Router) {
	r.Get("/api/v1/courses/{course_code}/analytics/insights", d.handleGetCourseInsights())
	r.Get("/api/v1/courses/{course_code}/analytics/cross-section", d.handleGetCrossSection())
	r.Post("/api/v1/courses/{course_code}/analytics/insights/dismiss", d.handleDismissInsightSignal())
	r.Post("/api/v1/courses/{course_code}/analytics/insights/refresh", d.handleRefreshInsights())
}
