package httpserver

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/lextures/lextures/server/internal/apierr"
	"github.com/lextures/lextures/server/internal/courseroles"
	"github.com/lextures/lextures/server/internal/repos/course"
)

type patchCoursePlagiarismBody struct {
	PlagiarismChecksEnabled     *bool    `json:"plagiarismChecksEnabled"`
	PlagiarismProvider          *string  `json:"plagiarismProvider"`
	PlagiarismAlertThresholdPct *float64 `json:"plagiarismAlertThresholdPct"`
}

// handleGetCoursePlagiarismSettings is GET /api/v1/courses/{course_code}/plagiarism-settings
func (d Deps) handleGetCoursePlagiarismSettings() http.HandlerFunc {
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
		if !d.requirePlagiarismWorkflow(w) {
			return
		}
		courseCode, viewer, ok := d.requireCourseAccess(w, r)
		if !ok {
			return
		}
		hasPerm, err := courseroles.UserHasPermission(r.Context(), d.Pool, viewer, "course:"+courseCode+":item:create")
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to verify permissions.")
			return
		}
		if !hasPerm {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "Forbidden.")
			return
		}
		settings, err := course.GetPlagiarismSettings(r.Context(), d.Pool, courseCode)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load settings.")
			return
		}
		if settings == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Course not found.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(settings)
	}
}

// handlePatchCoursePlagiarismSettings is PATCH /api/v1/courses/{course_code}/plagiarism-settings
func (d Deps) handlePatchCoursePlagiarismSettings() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if r.Method != http.MethodPatch {
			w.Header().Set("Allow", http.MethodPatch+","+http.MethodOptions)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		if !d.requirePlagiarismWorkflow(w) {
			return
		}
		courseCode, viewer, ok := d.requireCourseAccess(w, r)
		if !ok {
			return
		}
		hasPerm, err := courseroles.UserHasPermission(r.Context(), d.Pool, viewer, "course:"+courseCode+":item:create")
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to verify permissions.")
			return
		}
		if !hasPerm {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "Forbidden.")
			return
		}
		var body patchCoursePlagiarismBody
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		if body.PlagiarismProvider != nil {
			p := strings.ToLower(strings.TrimSpace(*body.PlagiarismProvider))
			switch p {
			case "":
				body.PlagiarismProvider = nil
			case "none", "turnitin", "copyleaks", "gptzero":
				body.PlagiarismProvider = &p
			default:
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid plagiarismProvider.")
				return
			}
		}
		settings, err := course.PatchPlagiarismSettings(r.Context(), d.Pool, courseCode, course.PlagiarismSettingsPatch{
			ChecksEnabled:     body.PlagiarismChecksEnabled,
			Provider:          body.PlagiarismProvider,
			AlertThresholdPct: body.PlagiarismAlertThresholdPct,
		})
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to update settings.")
			return
		}
		if settings == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Course not found.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(settings)
	}
}
