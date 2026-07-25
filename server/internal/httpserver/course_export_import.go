package httpserver

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/lextures/lextures/server/internal/apierr"
	"github.com/lextures/lextures/server/internal/repos/rbac"
	"github.com/lextures/lextures/server/internal/service/courseexportimport"
)

// handleCourseExportGet is GET /api/v1/courses/{course_code}/export.
// Returns a full course JSON backup (syllabus, structure, bodies, grading, enrollments).
func (d Deps) handleCourseExportGet() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet+","+http.MethodOptions)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}

		courseCode, viewer, ok := d.requireCourseAccess(w, r)
		if !ok {
			return
		}

		perm := "course:" + courseCode + ":item:create"
		canEdit, err := rbac.UserHasPermission(r.Context(), d.Pool, viewer, perm)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to verify permissions.")
			return
		}
		if !canEdit {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "You need permission to edit course modules to export.")
			return
		}

		bundle, err := courseexportimport.BuildExport(r.Context(), d.Pool, courseCode)
		if err != nil {
			if errors.Is(err, courseexportimport.ErrNotFound) {
				apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Course not found.")
				return
			}
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to build course export.")
			return
		}

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Header().Set("Content-Disposition", `attachment; filename="`+courseCode+`-course-export.json"`)
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(bundle)
	}
}
