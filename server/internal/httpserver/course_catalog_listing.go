package httpserver

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/lextures/lextures/server/internal/apierr"
	"github.com/lextures/lextures/server/internal/courseroles"
	"github.com/lextures/lextures/server/internal/currency"
	"github.com/lextures/lextures/server/internal/repos/course"
	"github.com/lextures/lextures/server/internal/telemetry"
)

type catalogListingBody struct {
	IsPublic          bool    `json:"isPublic"`
	Category          *string `json:"category"`
	DifficultyLevel   *string `json:"difficultyLevel"`
	Language          string  `json:"language"`
	PriceCents        int     `json:"priceCents"`
	PriceCurrency     string  `json:"priceCurrency"`
	Slug              string  `json:"slug"`
	MarketplaceListed *bool   `json:"marketplaceListed"`
}

// catalogListingOff writes 404 when neither the public catalog nor the in-app
// marketplace is enabled (plan 15.1 / MKT2).
func (d Deps) catalogListingOff(w http.ResponseWriter) bool {
	cfg := d.effectiveConfig()
	if !cfg.FFPublicCatalog && !cfg.FFCourseMarketplace {
		apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Course catalog settings are not enabled.")
		return true
	}
	return false
}

// handleGetCourseCatalogListing is GET /api/v1/courses/{course_code}/catalog-listing.
func (d Deps) handleGetCourseCatalogListing() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if d.catalogListingOff(w) {
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
// It lets a course owner/instructor publish the course to the public catalog (plan 15.1)
// and manage in-app marketplace listing (plan MKT2).
func (d Deps) handlePutCourseCatalogListing() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if d.catalogListingOff(w) {
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
		existing, err := course.GetCatalogListing(r.Context(), d.Pool, courseCode)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load catalog listing.")
			return
		}
		if existing == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Course not found.")
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
		priceCurrency := course.NormalizePriceCurrency(body.PriceCurrency)
		if body.PriceCurrency != "" && !course.ValidPriceCurrency(priceCurrency) {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Unsupported currency.")
			return
		}
		if err := currency.ValidateCatalogPrice(body.PriceCents, priceCurrency); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, err.Error())
			return
		}

		marketplaceListed := existing.MarketplaceListed
		if body.MarketplaceListed != nil {
			marketplaceListed = *body.MarketplaceListed
		}
		if marketplaceListed && existing.PublishState != "published" {
			apierr.WriteJSON(
				w,
				http.StatusUnprocessableEntity,
				apierr.CodeInvalidInput,
				"Publish the course before listing it in the marketplace.",
			)
			return
		}

		listing, err := course.SetCatalogListing(r.Context(), d.Pool, courseCode, course.CatalogListing{
			IsPublic:          body.IsPublic,
			Category:          body.Category,
			DifficultyLevel:   body.DifficultyLevel,
			Language:          body.Language,
			PriceCents:        body.PriceCents,
			PriceCurrency:     priceCurrency,
			Slug:              body.Slug,
			MarketplaceListed: marketplaceListed,
		})
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to update catalog listing.")
			return
		}
		if listing == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Course not found.")
			return
		}
		d.invalidateCatalogCache(r.Context())
		telemetry.RecordMarketplaceListingSaved(listing.MarketplaceListed, course.IsFree(listing.PriceCents))
		writeJSON(w, http.StatusOK, map[string]any{"listing": listing})
	}
}
