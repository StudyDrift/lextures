package httpserver

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/lextures/lextures/server/internal/apierr"
	"github.com/lextures/lextures/server/internal/courseroles"
	"github.com/lextures/lextures/server/internal/repos/course"
)

type catalogListingBody struct {
	IsPublic        bool    `json:"isPublic"`
	Category        *string `json:"category"`
	DifficultyLevel *string `json:"difficultyLevel"`
	Language        string  `json:"language"`
	PriceCents      int     `json:"priceCents"`
	Slug            string  `json:"slug"`
}

// handleGetCourseCatalogListing is GET /api/v1/courses/{course_code}/catalog-listing.
func (d Deps) handleGetCourseCatalogListing() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if d.publicCatalogOff(w) {
			return
		}
		courseCode, viewer, ok := d.requireCourseAccess(w, r)
		if !ok {
			return
		}
		_ = viewer
		listing, err := course.GetCatalogListing(r.Context(), d.Pool, courseCode)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load catalog listing.")
			return
		}
		if listing == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Course not found.")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"listing": listing})
	}
}

// handlePutCourseCatalogListing is PUT /api/v1/courses/{course_code}/catalog-listing.
// It lets a course owner/instructor publish the course to the public catalog (plan 15.1).
func (d Deps) handlePutCourseCatalogListing() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if d.publicCatalogOff(w) {
			return
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		courseCode, viewer, ok := d.requireCourseAccess(w, r)
		if !ok {
			return
		}
		canEdit, err := courseroles.UserHasPermission(r.Context(), d.Pool, viewer, "course:"+courseCode+":item:create")
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to verify permissions.")
			return
		}
		if !canEdit {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "You do not have permission to edit this course.")
			return
		}
		var body catalogListingBody
		if decErr := json.NewDecoder(r.Body).Decode(&body); decErr != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		if body.DifficultyLevel != nil {
			lvl := strings.TrimSpace(*body.DifficultyLevel)
			if lvl == "" {
				body.DifficultyLevel = nil
			} else if !course.ValidDifficultyLevel(lvl) {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid difficulty level.")
				return
			} else {
				body.DifficultyLevel = &lvl
			}
		}
		if body.Category != nil {
			cat := strings.TrimSpace(*body.Category)
			if cat == "" {
				body.Category = nil
			} else {
				body.Category = &cat
			}
		}
		if body.PriceCents < 0 {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Price cannot be negative.")
			return
		}
		listing, err := course.SetCatalogListing(r.Context(), d.Pool, courseCode, course.CatalogListing{
			IsPublic:        body.IsPublic,
			Category:        body.Category,
			DifficultyLevel: body.DifficultyLevel,
			Language:        body.Language,
			PriceCents:      body.PriceCents,
			Slug:            body.Slug,
		})
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to update catalog listing.")
			return
		}
		if listing == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Course not found.")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"listing": listing})
	}
}
