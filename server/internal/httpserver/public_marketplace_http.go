package httpserver

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/lextures/lextures/server/internal/apierr"
	"github.com/lextures/lextures/server/internal/objectcache"
	repoCourse "github.com/lextures/lextures/server/internal/repos/course"
	"github.com/lextures/lextures/server/internal/service/catalogsearch"
	"github.com/lextures/lextures/server/internal/service/coursereviews"
	"github.com/lextures/lextures/server/internal/telemetry"
)

// publicMarketplaceListResult is the anonymous listing payload (no ownership).
type publicMarketplaceListResult struct {
	Courses    []publicMarketplaceCourseJSON `json:"courses"`
	Total      int                           `json:"total"`
	NextCursor string                        `json:"nextCursor"`
}

// publicMarketplaceCourseJSON mirrors MarketplaceCourse without the owned field (plan MKT7).
type publicMarketplaceCourseJSON struct {
	ID              string   `json:"id"`
	Slug            string   `json:"slug"`
	CourseCode      string   `json:"courseCode"`
	Title           string   `json:"title"`
	Description     string   `json:"description"`
	HeroImageURL    *string  `json:"heroImageUrl"`
	Category        *string  `json:"category"`
	Level           *string  `json:"level"`
	Language        string   `json:"language"`
	PriceCents      int      `json:"priceCents"`
	PriceCurrency   string   `json:"priceCurrency"`
	ListPriceCents  *int     `json:"listPriceCents"`
	EnrollmentCount int      `json:"enrollmentCount"`
	AverageRating   *float64 `json:"averageRating"`
	RatingCount     int      `json:"ratingCount"`
	InstructorName  *string  `json:"instructorName"`
	CreatedAt       string   `json:"createdAt"`
}

func toPublicMarketplaceCourse(c repoCourse.MarketplaceCourse) publicMarketplaceCourseJSON {
	return publicMarketplaceCourseJSON{
		ID:              c.ID,
		Slug:            c.Slug,
		CourseCode:      c.CourseCode,
		Title:           c.Title,
		Description:     c.Description,
		HeroImageURL:    c.HeroImageURL,
		Category:        c.Category,
		Level:           c.Level,
		Language:        c.Language,
		PriceCents:      c.PriceCents,
		PriceCurrency:   c.PriceCurrency,
		ListPriceCents:  c.ListPriceCents,
		EnrollmentCount: c.EnrollmentCount,
		AverageRating:   c.AverageRating,
		RatingCount:     c.RatingCount,
		InstructorName:  c.InstructorName,
		CreatedAt:       c.CreatedAt,
	}
}

func toPublicMarketplaceCourses(in []repoCourse.MarketplaceCourse) []publicMarketplaceCourseJSON {
	out := make([]publicMarketplaceCourseJSON, len(in))
	for i := range in {
		out[i] = toPublicMarketplaceCourse(in[i])
	}
	return out
}

// marketplaceCourseToPublicCatalog adapts a marketplace course for JSON-LD builders.
func marketplaceCourseToPublicCatalog(c repoCourse.MarketplaceCourse) repoCourse.PublicCatalogCourse {
	return repoCourse.PublicCatalogCourse{
		ID:              c.ID,
		Slug:            c.Slug,
		CourseCode:      c.CourseCode,
		Title:           c.Title,
		Description:     c.Description,
		HeroImageURL:    c.HeroImageURL,
		Category:        c.Category,
		DifficultyLevel: c.Level,
		Language:        c.Language,
		PriceCents:      c.PriceCents,
		EnrollmentCount: c.EnrollmentCount,
		AverageRating:   c.AverageRating,
		RatingCount:     c.RatingCount,
		InstructorName:  c.InstructorName,
		CreatedAt:       c.CreatedAt,
	}
}

// registerPublicMarketplaceRoutes wires unauthenticated marketplace read endpoints (plan MKT7).
func (d Deps) registerPublicMarketplaceRoutes(r chi.Router) {
	r.Get("/api/v1/public/marketplace/courses", d.handlePublicMarketplaceCourseList())
	r.Get("/api/v1/public/marketplace/categories", d.handlePublicMarketplaceCategories())
	r.Get("/api/v1/public/marketplace/courses/{slug}", d.handlePublicMarketplaceCourseDetail())
	r.Get("/api/v1/public/marketplace/courses/{slug}/reviews", d.handlePublicMarketplaceReviews())
}

func (d Deps) publicMarketplaceOff(w http.ResponseWriter) bool {
	if !d.effectiveConfig().FFCourseMarketplace {
		apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Marketplace is not enabled.")
		return true
	}
	return false
}

func (d Deps) parsePublicMarketplaceFilter(w http.ResponseWriter, r *http.Request) (repoCourse.MarketplaceFilter, bool) {
	qp := r.URL.Query()
	f := repoCourse.MarketplaceFilter{
		Q:        strings.TrimSpace(qp.Get("q")),
		Category: strings.TrimSpace(qp.Get("category")),
		Language: strings.TrimSpace(qp.Get("language")),
	}
	if lvl := strings.TrimSpace(qp.Get("level")); lvl != "" {
		if !repoCourse.ValidDifficultyLevel(lvl) {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid level filter.")
			return f, false
		}
		f.Level = lvl
	}
	if sort := strings.TrimSpace(qp.Get("sort")); sort != "" {
		if !repoCourse.ValidCatalogSort(sort) {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid sort.")
			return f, false
		}
		f.Sort = sort
	}
	if freeOnly := strings.TrimSpace(qp.Get("free_only")); freeOnly != "" {
		switch strings.ToLower(freeOnly) {
		case "1", "true", "yes":
			f.FreeOnly = true
		case "0", "false", "no":
			f.FreeOnly = false
		default:
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid free_only.")
			return f, false
		}
	}
	if pm := strings.TrimSpace(qp.Get("price_max")); pm != "" {
		v, err := strconv.Atoi(pm)
		if err != nil || v < 0 {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid price_max.")
			return f, false
		}
		f.PriceMax = &v
	}
	if lim := strings.TrimSpace(qp.Get("limit")); lim != "" {
		v, err := strconv.Atoi(lim)
		if err != nil || v <= 0 {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid limit.")
			return f, false
		}
		f.Limit = v
	}
	off, err := repoCourse.DecodeCatalogCursor(qp.Get("cursor"))
	if err != nil {
		apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid cursor.")
		return f, false
	}
	f.Offset = off
	return f, true
}

func (d Deps) handlePublicMarketplaceCourseList() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if d.publicMarketplaceOff(w) {
			return
		}
		f, ok := d.parsePublicMarketplaceFilter(w, r)
		if !ok {
			return
		}

		cacheKey := objectcache.MarketplacePageKey(f) + ":public"
		var cached publicMarketplaceListResult
		if c := d.objectCache(); c != nil {
			if hit, _ := c.GetJSON(r.Context(), cacheKey, objectcache.ResourceCatalogPage, &cached); hit {
				telemetry.RecordPublicMarketplaceList()
				publicCatalogCacheHeaders(w)
				writeJSONWithETag(w, r, http.StatusOK, cached)
				return
			}
		}

		courses, total, next, err := repoCourse.ListMarketplaceCourses(r.Context(), d.Pool, f)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to search marketplace.")
			return
		}
		res := publicMarketplaceListResult{
			Courses:    toPublicMarketplaceCourses(courses),
			Total:      total,
			NextCursor: next,
		}
		if c := d.objectCache(); c != nil {
			_ = c.SetJSON(r.Context(), cacheKey, res, cacheTTLCatalogPage)
		}
		telemetry.RecordPublicMarketplaceList()
		publicCatalogCacheHeaders(w)
		writeJSONWithETag(w, r, http.StatusOK, res)
	}
}

func (d Deps) handlePublicMarketplaceCategories() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if d.publicMarketplaceOff(w) {
			return
		}
		cats, err := repoCourse.ListMarketplaceCategories(r.Context(), d.Pool)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to list categories.")
			return
		}
		publicCatalogCacheHeaders(w)
		writeJSONWithETag(w, r, http.StatusOK, map[string]any{"categories": cats})
	}
}

func (d Deps) handlePublicMarketplaceCourseDetail() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if d.publicMarketplaceOff(w) {
			return
		}
		slug := strings.TrimSpace(chi.URLParam(r, "slug"))
		course, err := repoCourse.GetMarketplaceCourseBySlug(r.Context(), d.Pool, slug)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load course.")
			return
		}
		if course == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Course not found.")
			return
		}

		included := repoCourse.MarketplaceWhatsIncluded{}
		if courseID, parseErr := uuid.Parse(course.ID); parseErr == nil {
			if wi, inclErr := repoCourse.GetMarketplaceWhatsIncluded(r.Context(), d.Pool, courseID); inclErr == nil {
				included = wi
			}
		}

		// JSON-LD URLs point at the public www course pages (plan MKT7 OQ#2 / MKT10).
		jsonLdBase := marketplaceJSONLDBaseURL(r)
		jsonLd := catalogsearch.BuildCourseJSONLDAt(
			marketplaceCourseToPublicCatalog(*course),
			jsonLdBase,
			"/courses/",
		)

		publicCatalogCacheHeaders(w)
		writeJSONWithETag(w, r, http.StatusOK, map[string]any{
			"course":        toPublicMarketplaceCourse(*course),
			"whatsIncluded": included,
			"jsonLd":        jsonLd,
		})
	}
}

// handlePublicMarketplaceReviews lists reviews for a marketplace-listed course by slug.
// Sibling of the public catalog reviews endpoint so marketplace-only (non-is_public)
// courses still resolve (plan MKT7 FR-10).
func (d Deps) handlePublicMarketplaceReviews() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if d.publicMarketplaceOff(w) || d.courseReviewsFeatureOff(w) {
			return
		}
		slug := strings.TrimSpace(chi.URLParam(r, "slug"))
		course, err := repoCourse.GetMarketplaceCourseBySlug(r.Context(), d.Pool, slug)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load course.")
			return
		}
		if course == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Course not found.")
			return
		}
		courseID, err := uuid.Parse(course.ID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Invalid course id.")
			return
		}
		cursor := strings.TrimSpace(r.URL.Query().Get("cursor"))
		limit := 10
		if v := strings.TrimSpace(r.URL.Query().Get("limit")); v != "" {
			if n, err := strconv.Atoi(v); err == nil {
				limit = n
			}
		}
		result, err := coursereviews.List(r.Context(), d.Pool, courseID, cursor, limit)
		if err != nil {
			if errors.Is(err, coursereviews.ErrNotFound) {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid cursor.")
				return
			}
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load reviews.")
			return
		}
		reviews := make([]map[string]any, 0, len(result.Reviews))
		for i := range result.Reviews {
			reviews = append(reviews, reviewToJSON(result.Reviews[i]))
		}
		out := map[string]any{
			"summary": summaryToJSON(result.Summary),
			"reviews": reviews,
		}
		if result.NextCursor != "" {
			out["nextCursor"] = result.NextCursor
		}
		publicCatalogCacheHeaders(w)
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(out)
	}
}

// marketplaceJSONLDBaseURL prefers an explicit marketing-site origin when the
// request hits the API host (self.lextures.com); falls back to the request origin.
func marketplaceJSONLDBaseURL(r *http.Request) string {
	if origin := strings.TrimSpace(r.Header.Get("X-Marketing-Origin")); origin != "" {
		return strings.TrimRight(origin, "/")
	}
	// Default public marketing site for Course JSON-LD on www pages.
	return "https://lextures.com"
}
