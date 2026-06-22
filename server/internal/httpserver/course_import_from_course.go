package httpserver

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/lextures/lextures/server/internal/apierr"
	"github.com/lextures/lextures/server/internal/repos/course"
	"github.com/lextures/lextures/server/internal/courseroles"
	"github.com/lextures/lextures/server/internal/repos/enrollment"
	"github.com/lextures/lextures/server/internal/repos/rbac"
	"github.com/lextures/lextures/server/internal/service/coursecopy"
)

// handlePostCourseImportFromCourse is POST /api/v1/courses/import/from-course.
func (d Deps) handlePostCourseImportFromCourse() http.HandlerFunc {
	type body struct {
		SourceCourseCode string              `json:"sourceCourseCode"`
		Title            string              `json:"title"`
		Description      *string             `json:"description"`
		Include          coursecopy.Include  `json:"include"`
	}
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

		allowed, err := rbac.UserHasPermission(r.Context(), d.Pool, userID, "global:app:course:create")
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to verify permissions.")
			return
		}
		if !allowed {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "You do not have permission to create courses.")
			return
		}

		var req body
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}

		sourceCode := strings.TrimSpace(req.SourceCourseCode)
		title := strings.TrimSpace(req.Title)
		if sourceCode == "" {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "sourceCourseCode is required.")
			return
		}
		if title == "" {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "title is required.")
			return
		}

		hasAccess, err := enrollment.UserHasAccess(r.Context(), d.Pool, sourceCode, userID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to verify course access.")
			return
		}
		if !hasAccess {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Source course not found.")
			return
		}

		canImport, err := courseroles.UserHasPermission(r.Context(), d.Pool, userID, "course:"+sourceCode+":item:create")
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to verify permissions.")
			return
		}
		if !canImport {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "You need permission to edit the source course to copy from it.")
			return
		}

		sourceID, err := course.GetIDByCourseCode(r.Context(), d.Pool, sourceCode)
		if err != nil || sourceID == nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load source course.")
			return
		}

		description := ""
		if req.Description != nil {
			description = strings.TrimSpace(*req.Description)
		}
		if description == "" {
			src, err := course.GetPublicByCourseCode(r.Context(), d.Pool, sourceCode)
			if err != nil {
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load source course.")
				return
			}
			if src != nil {
				description = strings.TrimSpace(src.Description)
			}
		}

		created, err := course.CreateCourse(r.Context(), d.Pool, userID, title, description, "traditional", nil, nil, nil)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to create course.")
			return
		}

		targetID, err := course.GetIDByCourseCode(r.Context(), d.Pool, created.CourseCode)
		if err != nil || targetID == nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load new course.")
			return
		}

		cfg := d.effectiveConfig()
		if err := coursecopy.CopyFromCourse(r.Context(), d.Pool, coursecopy.Options{
			SourceCourseID:   *sourceID,
			TargetCourseID:   *targetID,
			SourceCourseCode: sourceCode,
			TargetCourseCode: created.CourseCode,
			Include:          req.Include,
			FilesRoot:        cfg.CourseFilesRoot,
			ActorUserID:      userID,
		}); err != nil {
			d.notifyCourses(userID)
			d.pushNotificationService().EnqueueCourseCopyImportFailed(
				r.Context(), userID, title, created.CourseCode, err.Error(),
			)
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, err.Error())
			return
		}

		out, err := course.GetPublicByCourseCode(r.Context(), d.Pool, created.CourseCode)
		if err != nil || out == nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Course created but could not be loaded.")
			return
		}

		d.notifyCourses(userID)
		d.pushNotificationService().EnqueueCourseCopyImported(r.Context(), userID, title, created.CourseCode)

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(out)
	}
}