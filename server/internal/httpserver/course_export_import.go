package httpserver

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/lextures/lextures/server/internal/apierr"
	"github.com/lextures/lextures/server/internal/models/courseexport"
	"github.com/lextures/lextures/server/internal/repos/course"
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

		cfg := d.effectiveConfig()
		blobOpts := courseexportimport.BlobOptions{
			FilesRoot: cfg.CourseFilesRoot,
			Storage:   d.Storage,
		}
		bundle, err := courseexportimport.BuildExport(r.Context(), d.Pool, courseCode, blobOpts)
		if err != nil {
			if errors.Is(err, courseexportimport.ErrNotFound) {
				apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Course not found.")
				return
			}
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to build course export.")
			return
		}

		// Side effect: successful export stamps launch.backup-export (CC.6).
		_ = course.StampLastExportByCourseCode(r.Context(), d.Pool, courseCode)

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Header().Set("Content-Disposition", `attachment; filename="`+courseCode+`-course-export.json"`)
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(bundle)
	}
}

// handleCourseImportPost is POST /api/v1/courses/{course_code}/import.
// Body: { "mode": "erase"|"mergeAdd"|"overwrite", "export": <CourseExportV1> }.
func (d Deps) handleCourseImportPost() http.HandlerFunc {
	type body struct {
		Mode   courseexport.CourseImportMode  `json:"mode"`
		Export *courseexportimport.Bundle     `json:"export"`
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
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "You need permission to edit course modules to import.")
			return
		}

		var req body
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		if req.Export == nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "export is required.")
			return
		}

		cfg := d.effectiveConfig()
		blobOpts := courseexportimport.BlobOptions{
			FilesRoot: cfg.CourseFilesRoot,
			Storage:   d.Storage,
		}
		err = courseexportimport.ApplyImport(r.Context(), d.Pool, courseCode, req.Mode, req.Export, nil, blobOpts)
		if err != nil {
			if errors.Is(err, courseexportimport.ErrNotFound) {
				apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Course not found.")
				return
			}
			if courseexportimport.IsInvalidInput(err) {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, courseexportimport.InvalidInputMessage(err))
				return
			}
			slog.Error("course import failed", "course_code", courseCode, "mode", req.Mode, "err", err)
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to import course.")
			return
		}

		broadcastStructureChanged(courseCode)
		d.notifyCourses(viewer)
		w.WriteHeader(http.StatusNoContent)
	}
}
