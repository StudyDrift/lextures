package httpserver

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/lextures/lextures/server/internal/apierr"
	"github.com/lextures/lextures/server/internal/objectcache"
	repoBilling "github.com/lextures/lextures/server/internal/repos/billing"
	repoCourse "github.com/lextures/lextures/server/internal/repos/course"
	"github.com/lextures/lextures/server/internal/telemetry"
)

// marketplaceListResult is the cached listing payload (ownership applied after cache).
type marketplaceListResult struct {
	Courses    []repoCourse.MarketplaceCourse `json:"courses"`
	Total      int                            `json:"total"`
	NextCursor string                         `json:"nextCursor"`
}

// registerMarketplaceCourseRoutes wires authenticated in-app course marketplace
// read endpoints (plan MKT3). Distinct from plugin marketplace_http.go routes
// under /api/v1/marketplace/apps.
func (d Deps) registerMarketplaceCourseRoutes(r chi.Router) {
	r.Get("/api/v1/marketplace/courses", d.handleMarketplaceCourseList())
	r.Get("/api/v1/marketplace/categories", d.handleMarketplaceCategories())
	r.Get("/api/v1/marketplace/courses/{slug}", d.handleMarketplaceCourseDetail())
	d.registerMarketplacePurchaseRoutes(r)
}

func (d Deps) handleMarketplaceCourseList() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		if d.courseMarketplaceOff(w) {
			return
		}
		qp := r.URL.Query()
		f := repoCourse.MarketplaceFilter{
			Q:        strings.TrimSpace(qp.Get("q")),
			Category: strings.TrimSpace(qp.Get("category")),
			Language: strings.TrimSpace(qp.Get("language")),
		}
		if lvl := strings.TrimSpace(qp.Get("level")); lvl != "" {
			if !repoCourse.ValidDifficultyLevel(lvl) {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid level filter.")
				return
			}
			f.Level = lvl
		}
		if sort := strings.TrimSpace(qp.Get("sort")); sort != "" {
			if !repoCourse.ValidCatalogSort(sort) {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid sort.")
				return
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
				return
			}
		}
		if pm := strings.TrimSpace(qp.Get("price_max")); pm != "" {
			v, err := strconv.Atoi(pm)
			if err != nil || v < 0 {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid price_max.")
				return
			}
			f.PriceMax = &v
		}
		if lim := strings.TrimSpace(qp.Get("limit")); lim != "" {
			v, err := strconv.Atoi(lim)
			if err != nil || v <= 0 {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid limit.")
				return
			}
			f.Limit = v
		}
		off, err := repoCourse.DecodeCatalogCursor(qp.Get("cursor"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid cursor.")
			return
		}
		f.Offset = off

		cacheKey := objectcache.MarketplacePageKey(f)
		var cached marketplaceListResult
		if c := d.objectCache(); c != nil {
			if hit, _ := c.GetJSON(r.Context(), cacheKey, objectcache.ResourceCatalogPage, &cached); hit {
				d.applyMarketplaceOwnership(r, userID, cached.Courses)
				telemetry.RecordMarketplaceStorefrontView()
				writeJSON(w, http.StatusOK, cached)
				return
			}
		}

		courses, total, next, err := repoCourse.ListMarketplaceCourses(r.Context(), d.Pool, f)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to search marketplace.")
			return
		}
		res := marketplaceListResult{Courses: courses, Total: total, NextCursor: next}
		if c := d.objectCache(); c != nil {
			_ = c.SetJSON(r.Context(), cacheKey, res, cacheTTLCatalogPage)
		}
		d.applyMarketplaceOwnership(r, userID, res.Courses)
		telemetry.RecordMarketplaceStorefrontView()
		if f.Category != "" || f.Level != "" || f.Language != "" || f.FreeOnly || f.PriceMax != nil || f.Q != "" {
			telemetry.RecordMarketplaceFacetUsage()
		}
		writeJSON(w, http.StatusOK, res)
	}
}

func (d Deps) handleMarketplaceCategories() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, ok := d.meUserID(w, r); !ok {
			return
		}
		if d.courseMarketplaceOff(w) {
			return
		}
		cats, err := repoCourse.ListMarketplaceCategories(r.Context(), d.Pool)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to list categories.")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"categories": cats})
	}
}

func (d Deps) handleMarketplaceCourseDetail() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		if d.courseMarketplaceOff(w) {
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

		owned := false
		if courseID, parseErr := uuid.Parse(course.ID); parseErr == nil {
			if ok, accessErr := repoBilling.MarketplaceAccess(r.Context(), d.Pool, userID, courseID); accessErr == nil {
				owned = ok
			}
			course.Owned = owned
			included, inclErr := repoCourse.GetMarketplaceWhatsIncluded(r.Context(), d.Pool, courseID)
			if inclErr != nil {
				included = repoCourse.MarketplaceWhatsIncluded{}
			}
			telemetry.RecordMarketplaceDetailView(owned)
			writeJSON(w, http.StatusOK, map[string]any{
				"course":         course,
				"owned":          owned,
				"priceCents":     course.PriceCents,
				"priceCurrency":  course.PriceCurrency,
				"listPriceCents": course.ListPriceCents,
				"whatsIncluded":  included,
				"rating": map[string]any{
					"average": course.AverageRating,
					"count":   course.RatingCount,
				},
			})
			return
		}
		telemetry.RecordMarketplaceDetailView(false)
		writeJSON(w, http.StatusOK, map[string]any{
			"course":         course,
			"owned":          false,
			"priceCents":     course.PriceCents,
			"priceCurrency":  course.PriceCurrency,
			"listPriceCents": course.ListPriceCents,
			"whatsIncluded":  repoCourse.MarketplaceWhatsIncluded{},
			"rating": map[string]any{
				"average": course.AverageRating,
				"count":   course.RatingCount,
			},
		})
	}
}

// applyMarketplaceOwnership sets Owned on each course. Ownership query failures
// degrade to owned=false so browsing is never blocked (plan MKT3 NFR Reliability).
func (d Deps) applyMarketplaceOwnership(r *http.Request, userID uuid.UUID, courses []repoCourse.MarketplaceCourse) {
	if len(courses) == 0 || d.Pool == nil {
		return
	}
	ids := make([]uuid.UUID, 0, len(courses))
	for _, c := range courses {
		id, err := uuid.Parse(c.ID)
		if err != nil {
			continue
		}
		ids = append(ids, id)
	}
	owned, err := repoBilling.OwnedCourseIDs(r.Context(), d.Pool, userID, ids)
	if err != nil {
		return
	}
	for i := range courses {
		id, err := uuid.Parse(courses[i].ID)
		if err != nil {
			continue
		}
		if _, ok := owned[id]; ok {
			courses[i].Owned = true
		}
	}
}
