package httpserver

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/lextures/lextures/server/internal/apierr"
	repoCourse "github.com/lextures/lextures/server/internal/repos/course"
	"github.com/lextures/lextures/server/internal/service/catalogsearch"
)

// registerPublicCatalogRoutes wires the unauthenticated public course catalog
// (plan 15.1). All endpoints are flag-gated by FFPublicCatalog and return 404
// when the feature is off (rollout rollback path).
func (d Deps) registerPublicCatalogRoutes(r chi.Router) {
	r.Get("/api/v1/public/catalog/courses", d.handlePublicCatalogList())
	r.Get("/api/v1/public/catalog/categories", d.handlePublicCatalogCategories())
	r.Get("/api/v1/public/catalog/courses/{slug}", d.handlePublicCatalogDetail())
	r.Get("/api/v1/public/catalog/courses/{slug}/reviews", d.handlePublicCatalogReviews())
	r.Get("/api/v1/internal/catalog/courses/{slug}/json-ld", d.handlePublicCatalogJSONLD())
}

func (d Deps) publicCatalogOff(w http.ResponseWriter) bool {
	if !d.effectiveConfig().FFPublicCatalog {
		apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Public course catalog is not enabled.")
		return true
	}
	return false
}

// publicCatalogCacheHeaders sets the 60-second CDN cache headers required for the
// public catalog (FR-9 / §9). Safe for anonymous, non-personalised responses.
func publicCatalogCacheHeaders(w http.ResponseWriter) {
	w.Header().Set("Cache-Control", "public, max-age=60, s-maxage=60")
	w.Header().Set("Vary", "Accept-Encoding")
}

func (d Deps) handlePublicCatalogList() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if d.publicCatalogOff(w) {
			return
		}
		qp := r.URL.Query()
		f := repoCourse.PublicCatalogFilter{
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

		svc := catalogsearch.New(d.Pool)
		res, err := svc.Search(r.Context(), f)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to search catalog.")
			return
		}
		publicCatalogCacheHeaders(w)
		writeJSON(w, http.StatusOK, res)
	}
}

func (d Deps) handlePublicCatalogCategories() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if d.publicCatalogOff(w) {
			return
		}
		svc := catalogsearch.New(d.Pool)
		cats, err := svc.Categories(r.Context())
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to list categories.")
			return
		}
		publicCatalogCacheHeaders(w)
		writeJSON(w, http.StatusOK, map[string]any{"categories": cats})
	}
}

func (d Deps) handlePublicCatalogDetail() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if d.publicCatalogOff(w) {
			return
		}
		slug := strings.TrimSpace(chi.URLParam(r, "slug"))
		svc := catalogsearch.New(d.Pool)
		c, err := svc.CourseBySlug(r.Context(), slug)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load course.")
			return
		}
		if c == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Course not found.")
			return
		}
		publicCatalogCacheHeaders(w)
		writeJSON(w, http.StatusOK, map[string]any{
			"course": c,
			"jsonLd": catalogsearch.BuildCourseJSONLD(*c, requestBaseURL(r)),
		})
	}
}

// handlePublicCatalogJSONLD serves only the Schema.org JSON-LD document for a
// course, for use by the SSR layer (§9 internal endpoint).
func (d Deps) handlePublicCatalogJSONLD() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if d.publicCatalogOff(w) {
			return
		}
		slug := strings.TrimSpace(chi.URLParam(r, "slug"))
		svc := catalogsearch.New(d.Pool)
		c, err := svc.CourseBySlug(r.Context(), slug)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load course.")
			return
		}
		if c == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Course not found.")
			return
		}
		publicCatalogCacheHeaders(w)
		writeJSON(w, http.StatusOK, catalogsearch.BuildCourseJSONLD(*c, requestBaseURL(r)))
	}
}

// requestBaseURL reconstructs the public origin from the incoming request,
// honouring the X-Forwarded-Proto header set by upstream proxies/CDN.
func requestBaseURL(r *http.Request) string {
	if r.Host == "" {
		return ""
	}
	scheme := "https"
	if proto := r.Header.Get("X-Forwarded-Proto"); proto != "" {
		scheme = proto
	} else if r.TLS == nil {
		scheme = "http"
	}
	return scheme + "://" + r.Host
}
