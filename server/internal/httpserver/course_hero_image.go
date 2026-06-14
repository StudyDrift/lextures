package httpserver

import (
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/lextures/lextures/server/internal/apierr"
	"github.com/lextures/lextures/server/internal/repos/course"
	"github.com/lextures/lextures/server/internal/repos/enrollment"
	"github.com/lextures/lextures/server/internal/repos/rbac"
)

func validHeroImageURL(raw string) bool {
	s := strings.TrimSpace(raw)
	if s == "" {
		return false
	}
	if strings.HasPrefix(s, "/api/v1/courses/") && strings.Contains(s, "/course-files/") && strings.HasSuffix(s, "/content") {
		return true
	}
	u, err := url.Parse(s)
	if err != nil {
		return false
	}
	switch strings.ToLower(u.Scheme) {
	case "http", "https":
		return u.Host != ""
	default:
		return false
	}
}

// handlePutCourseHeroImage is PUT /api/v1/courses/{course_code}/hero-image
func (d Deps) handlePutCourseHeroImage() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if r.Method != http.MethodPut {
			w.Header().Set("Allow", http.MethodPut+","+http.MethodOptions)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		courseCode := chi.URLParam(r, "course_code")
		if courseCode == "" {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Missing course code.")
			return
		}
		userID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		if d.Pool == nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Server misconfiguration.")
			return
		}
		hasAccess, err := enrollment.UserHasAccess(r.Context(), d.Pool, courseCode, userID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to verify course access.")
			return
		}
		if !hasAccess {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Course not found.")
			return
		}
		perm := "course:" + courseCode + ":item:create"
		hasPerm, err := rbac.UserHasPermission(r.Context(), d.Pool, userID, perm)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to verify permissions.")
			return
		}
		if !hasPerm {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "You do not have permission for this action.")
			return
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid request body.")
			return
		}
		_ = r.Body.Close()

		var raw map[string]json.RawMessage
		if err := json.Unmarshal(body, &raw); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}

		patch := course.HeroImagePatch{}
		if v, ok := raw["imageUrl"]; ok {
			patch.UpdateImageURL = true
			if string(v) == "null" {
				patch.ImageURL = nil
			} else {
				var imageURL string
				if err := json.Unmarshal(v, &imageURL); err != nil {
					apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid imageUrl.")
					return
				}
				imageURL = strings.TrimSpace(imageURL)
				if imageURL == "" {
					apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "imageUrl is required.")
					return
				}
				if !validHeroImageURL(imageURL) {
					apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid imageUrl.")
					return
				}
				patch.ImageURL = &imageURL
			}
		}
		if v, ok := raw["objectPosition"]; ok {
			patch.UpdateObjectPosition = true
			if string(v) == "null" {
				patch.ObjectPosition = nil
			} else {
				var objectPosition string
				if err := json.Unmarshal(v, &objectPosition); err != nil {
					apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid objectPosition.")
					return
				}
				objectPosition = strings.TrimSpace(objectPosition)
				if objectPosition == "" {
					patch.ObjectPosition = nil
				} else if !course.ValidHeroObjectPosition(objectPosition) {
					apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid objectPosition.")
					return
				} else {
					patch.ObjectPosition = &objectPosition
				}
			}
		}
		if !patch.UpdateImageURL && !patch.UpdateObjectPosition {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Provide imageUrl and/or objectPosition.")
			return
		}

		out, err := course.SetHeroImage(r.Context(), d.Pool, courseCode, patch)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to update hero image.")
			return
		}
		if out == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Course not found.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(out)
	}
}