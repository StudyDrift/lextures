package httpserver

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/lextures/lextures/server/internal/apierr"
	"github.com/lextures/lextures/server/internal/repos/course"
	"github.com/lextures/lextures/server/internal/repos/enrollment"
	"github.com/lextures/lextures/server/internal/repos/masteryheatmap"
)

// handleCourseMasteryHeatmap serves GET /api/v1/courses/{course_code}/analytics/mastery-heatmap.
// Requires course staff (instructor) role.
func (d Deps) handleCourseMasteryHeatmap() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		userID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		courseCode := chi.URLParam(r, "course_code")
		ctx := r.Context()

		isStaff, err := enrollment.UserIsCourseStaff(ctx, d.Pool, courseCode, userID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to check course access.")
			return
		}
		if !isStaff {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "Forbidden.")
			return
		}

		crow, err := course.GetPublicByCourseCode(ctx, d.Pool, courseCode)
		if err != nil || crow == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Course not found.")
			return
		}
		courseID, err := uuid.Parse(crow.ID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Invalid course id.")
			return
		}

		result, err := masteryheatmap.HeatmapForCourse(ctx, d.Pool, courseID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load heatmap.")
			return
		}

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(result)
	}
}

// handleCourseMasteryHeatmapConceptDrillDown serves
// GET /api/v1/courses/{course_code}/analytics/mastery-heatmap/concepts/{concept_id}.
func (d Deps) handleCourseMasteryHeatmapConceptDrillDown() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		userID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		courseCode := chi.URLParam(r, "course_code")
		ctx := r.Context()

		isStaff, err := enrollment.UserIsCourseStaff(ctx, d.Pool, courseCode, userID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to check course access.")
			return
		}
		if !isStaff {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "Forbidden.")
			return
		}

		conceptID, err := uuid.Parse(chi.URLParam(r, "concept_id"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid concept_id.")
			return
		}

		crow, err := course.GetPublicByCourseCode(ctx, d.Pool, courseCode)
		if err != nil || crow == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Course not found.")
			return
		}
		courseID, err := uuid.Parse(crow.ID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Invalid course id.")
			return
		}

		students, err := masteryheatmap.DrillDownForConcept(ctx, d.Pool, courseID, conceptID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load drill-down.")
			return
		}
		if students == nil {
			students = []masteryheatmap.DrillDownStudent{}
		}

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{"students": students})
	}
}

// handleCourseEnrollmentMastery serves
// GET /api/v1/courses/{course_code}/enrollments/{enrollment_id}/mastery.
// Accessible by course staff or the enrolled student themselves.
func (d Deps) handleCourseEnrollmentMastery() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		userID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		courseCode := chi.URLParam(r, "course_code")
		enrollmentIDStr := chi.URLParam(r, "enrollment_id")
		ctx := r.Context()

		enrollmentID, err := uuid.Parse(enrollmentIDStr)
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid enrollment_id.")
			return
		}

		crow, err := course.GetPublicByCourseCode(ctx, d.Pool, courseCode)
		if err != nil || crow == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Course not found.")
			return
		}
		courseID, err := uuid.Parse(crow.ID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Invalid course id.")
			return
		}

		// Allow access if requester is staff or is the enrollment owner.
		isStaff, err := enrollment.UserIsCourseStaffByID(ctx, d.Pool, courseID, userID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to check course access.")
			return
		}
		if !isStaff {
			// Check if the requester owns this enrollment.
			ownEID, err := enrollment.GetStudentEnrollmentID(ctx, d.Pool, courseID, userID)
			if err != nil || ownEID == nil || *ownEID != enrollmentID {
				apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "Forbidden.")
				return
			}
		}

		row, err := masteryheatmap.StudentMastery(ctx, d.Pool, courseID, enrollmentID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Enrollment not found.")
				return
			}
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load mastery.")
			return
		}

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(row)
	}
}

// handleCourseMasteryHeatmapRefresh serves
// POST /api/v1/courses/{course_code}/analytics/mastery-heatmap/refresh.
func (d Deps) handleCourseMasteryHeatmapRefresh() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		userID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		courseCode := chi.URLParam(r, "course_code")
		ctx := r.Context()

		isStaff, err := enrollment.UserIsCourseStaff(ctx, d.Pool, courseCode, userID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to check course access.")
			return
		}
		if !isStaff {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "Forbidden.")
			return
		}

		if err := masteryheatmap.RefreshMaterializedView(ctx, d.Pool); err != nil {
			// Ignore error if view doesn't exist yet (graceful empty state).
			_ = err
		}

		w.WriteHeader(http.StatusNoContent)
	}
}
