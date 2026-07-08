package httpserver

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/lextures/lextures/server/internal/apierr"
	platformcourses "github.com/lextures/lextures/server/internal/repos/platformcourses"
)

func (d Deps) registerPlatformCoursesRoutes(r chi.Router) {
	r.Get("/api/v1/admin/courses", d.handleAdminCoursesSearch())
	r.Get("/api/v1/admin/courses/{courseId}/report", d.handleAdminCoursesReport())
	r.Post("/api/v1/admin/courses/{courseId}/access", d.handleAdminCoursesAccess())
}

func parsePlatformCoursesListParams(r *http.Request) platformcourses.ListParams {
	status := strings.TrimSpace(r.URL.Query().Get("status"))
	if status == "" {
		status = "open"
	}
	p := platformcourses.ListParams{
		Query:  strings.TrimSpace(r.URL.Query().Get("q")),
		Status: status,
	}
	if v := strings.TrimSpace(r.URL.Query().Get("page")); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			p.Page = n
		}
	}
	if v := strings.TrimSpace(r.URL.Query().Get("per_page")); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			p.PerPage = n
		}
	}
	if v := strings.TrimSpace(r.URL.Query().Get("perPage")); v != "" && p.PerPage == 0 {
		if n, err := strconv.Atoi(v); err == nil {
			p.PerPage = n
		}
	}
	return p
}

func (d Deps) handleAdminCoursesSearch() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		if _, ok := d.adminRbacUser(w, r); !ok {
			return
		}
		params := parsePlatformCoursesListParams(r)
		if params.Query == "" {
			writeJSON(w, http.StatusOK, platformcourses.ListResult{Items: []platformcourses.CourseRow{}})
			return
		}
		result, err := platformcourses.Search(r.Context(), d.Pool, params)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to search courses.")
			return
		}
		writeJSON(w, http.StatusOK, result)
	}
}

func (d Deps) handleAdminCoursesReport() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		if _, ok := d.adminRbacUser(w, r); !ok {
			return
		}
		courseID, err := uuid.Parse(strings.TrimSpace(chi.URLParam(r, "courseId")))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid course id.")
			return
		}
		report, err := platformcourses.CourseReport(r.Context(), d.Pool, courseID)
		if err != nil {
			if err == pgx.ErrNoRows {
				apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Course not found.")
				return
			}
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load course report.")
			return
		}
		writeJSON(w, http.StatusOK, report)
	}
}

func (d Deps) handleAdminCoursesAccess() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		actor, ok := d.adminRbacUser(w, r)
		if !ok {
			return
		}
		courseID, err := uuid.Parse(strings.TrimSpace(chi.URLParam(r, "courseId")))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid course id.")
			return
		}
		if err := platformcourses.LookupCourseID(r.Context(), d.Pool, courseID); err != nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Course not found.")
			return
		}
		if err := platformcourses.EnsureAdminAccess(r.Context(), d.Pool, courseID, actor); err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to grant course access.")
			return
		}
		report, err := platformcourses.CourseReport(r.Context(), d.Pool, courseID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load course.")
			return
		}
		writeJSON(w, http.StatusOK, report)
	}
}