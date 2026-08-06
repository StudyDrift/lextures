package httpserver

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/lextures/lextures/server/internal/apierr"
	"github.com/lextures/lextures/server/internal/repos/course"
	acsvc "github.com/lextures/lextures/server/internal/service/adaptivecontent"
)

func (d Deps) registerAdaptiveContentReportRoutes(r chi.Router) {
	r.Get("/api/v1/courses/{course_code}/adaptive-content/report", d.handleAdaptiveContentCourseReportGet())
	r.Get("/api/v1/courses/{course_code}/adaptive-content/report/export", d.handleAdaptiveContentCourseReportExport())
	r.Get("/api/v1/admin/adaptive-content/report", d.handleAdminAdaptiveContentReportGet())
	r.Get("/api/v1/admin/adaptive-content/report/export", d.handleAdminAdaptiveContentReportExport())
}

// handleAdaptiveContentCourseReportGet is GET .../adaptive-content/report (instructor, AC.9 FR-1).
func (d Deps) handleAdaptiveContentCourseReportGet() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		courseCode, _, ok := d.requireCourseItemCreate(w, r)
		if !ok {
			return
		}
		cid, err := course.GetIDByCourseCode(r.Context(), d.Pool, courseCode)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load course.")
			return
		}
		if cid == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Course not found.")
			return
		}
		report, err := acsvc.BuildCourseReport(r.Context(), d.Pool, *cid, courseCode)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load Adaptive Content report.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(report)
	}
}

// handleAdaptiveContentCourseReportExport is GET .../adaptive-content/report/export (CSV, AC.9 FR-3).
func (d Deps) handleAdaptiveContentCourseReportExport() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		courseCode, _, ok := d.requireCourseItemCreate(w, r)
		if !ok {
			return
		}
		cid, err := course.GetIDByCourseCode(r.Context(), d.Pool, courseCode)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load course.")
			return
		}
		if cid == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Course not found.")
			return
		}
		report, err := acsvc.BuildCourseReport(r.Context(), d.Pool, *cid, courseCode)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load Adaptive Content report.")
			return
		}
		w.Header().Set("Content-Type", "text/csv; charset=utf-8")
		w.Header().Set("Content-Disposition", `attachment; filename="adaptive-content-report-`+courseCode+`.csv"`)
		if err := acsvc.WriteCourseReportCSV(w, report); err != nil {
			// Headers may already be flushed; best-effort log via status if possible.
			return
		}
	}
}

// handleAdminAdaptiveContentReportGet is GET /api/v1/admin/adaptive-content/report (AC.9 FR-2).
func (d Deps) handleAdminAdaptiveContentReportGet() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, ok := d.adminRbacUser(w, r); !ok {
			return
		}
		report, err := acsvc.BuildAdminReport(r.Context(), d.Pool)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load Adaptive Content org report.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(report)
	}
}

// handleAdminAdaptiveContentReportExport is GET /api/v1/admin/adaptive-content/report/export (CSV).
func (d Deps) handleAdminAdaptiveContentReportExport() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, ok := d.adminRbacUser(w, r); !ok {
			return
		}
		report, err := acsvc.BuildAdminReport(r.Context(), d.Pool)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load Adaptive Content org report.")
			return
		}
		w.Header().Set("Content-Type", "text/csv; charset=utf-8")
		w.Header().Set("Content-Disposition", `attachment; filename="adaptive-content-org-report.csv"`)
		_ = acsvc.WriteAdminReportCSV(w, report)
	}
}
