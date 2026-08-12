package httpserver

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/lextures/lextures/server/internal/apierr"
	mcrepo "github.com/lextures/lextures/server/internal/repos/marketingcontent"
)

// registerMarketingRouteHintsRoutes wires the MC.13 route-hints admin surface
// (FR-12): a small CRUD table mapping app route prefixes to help articles, plus
// a preview endpoint that mirrors what the widget would resolve for a route, and
// the zero-result search-gaps report (FR-14) that feeds content planning.
func (d Deps) registerMarketingRouteHintsRoutes(r chi.Router) {
	r.Get("/api/v1/admin/marketing/route-hints", d.handleMarketingRouteHintsList())
	r.Post("/api/v1/admin/marketing/route-hints", d.handleMarketingRouteHintCreate())
	r.Delete("/api/v1/admin/marketing/route-hints/{id}", d.handleMarketingRouteHintDelete())
	r.Get("/api/v1/admin/marketing/route-hints/preview", d.handleMarketingRouteHintsPreview())
	r.Get("/api/v1/admin/marketing/search-gaps", d.handleMarketingSearchGaps())
}

func (d Deps) handleMarketingRouteHintsList() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, ok := d.marketingAccess(w, r, marketingView); !ok {
			return
		}
		items, err := mcrepo.ListRouteHints(r.Context(), d.Pool)
		if err != nil {
			apierr.WriteInternal(w, r, "Failed to list route hints.", err)
			return
		}
		writeJSON(w, 200, map[string]any{"items": items})
	}
}

func (d Deps) handleMarketingRouteHintCreate() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		actor, ok := d.marketingAccess(w, r, marketingAdmin)
		if !ok {
			return
		}
		var body struct {
			RoutePrefix string    `json:"routePrefix"`
			ArticleID   uuid.UUID `json:"articleId"`
			Position    *int      `json:"position"`
		}
		if !readMarketingJSON(w, r, &body) {
			return
		}
		prefix := strings.TrimSpace(body.RoutePrefix)
		if prefix == "" || !strings.HasPrefix(prefix, "/") || strings.ContainsAny(prefix, "?#") {
			apierr.WriteJSON(w, 400, apierr.CodeInvalidInput, "routePrefix must be a non-empty path starting with /.")
			return
		}
		if body.ArticleID == uuid.Nil {
			apierr.WriteJSON(w, 400, apierr.CodeInvalidInput, "articleId is required.")
			return
		}
		position := 100
		if body.Position != nil {
			position = *body.Position
		}
		tx, err := d.Pool.Begin(r.Context())
		if err != nil {
			apierr.WriteInternal(w, r, "Failed to open transaction.", err)
			return
		}
		defer func() { _ = tx.Rollback(r.Context()) }()
		hint, err := mcrepo.InsertRouteHint(r.Context(), tx, prefix, body.ArticleID, position, actor)
		if err != nil {
			writeMarketingError(w, r, err)
			return
		}
		if err := tx.Commit(r.Context()); err != nil {
			apierr.WriteInternal(w, r, "Failed to save route hint.", err)
			return
		}
		d.recordMarketingTaxonomyAudit(r, actor, "route_hint_create", hint.ID)
		writeJSON(w, 201, hint)
	}
}

func (d Deps) handleMarketingRouteHintDelete() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		actor, ok := d.marketingAccess(w, r, marketingAdmin)
		if !ok {
			return
		}
		id, err := uuid.Parse(chi.URLParam(r, "id"))
		if err != nil {
			apierr.WriteJSON(w, 400, apierr.CodeInvalidInput, "Invalid route hint id.")
			return
		}
		tx, err := d.Pool.Begin(r.Context())
		if err != nil {
			apierr.WriteInternal(w, r, "Failed to open transaction.", err)
			return
		}
		defer func() { _ = tx.Rollback(r.Context()) }()
		if err := mcrepo.DeleteRouteHint(r.Context(), tx, id); err != nil {
			writeMarketingError(w, r, err)
			return
		}
		if err := tx.Commit(r.Context()); err != nil {
			apierr.WriteInternal(w, r, "Failed to delete route hint.", err)
			return
		}
		d.recordMarketingTaxonomyAudit(r, actor, "route_hint_delete", id)
		w.WriteHeader(204)
	}
}

// handleMarketingRouteHintsPreview shows admins exactly what the widget would
// return for a route, including the tier that answered (FR-12).
func (d Deps) handleMarketingRouteHintsPreview() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, ok := d.marketingAccess(w, r, marketingAdmin); !ok {
			return
		}
		route := strings.TrimSpace(r.URL.Query().Get("route"))
		if route == "" {
			apierr.WriteJSON(w, 400, apierr.CodeInvalidInput, "route is required.")
			return
		}
		articles, err := d.marketingService().ContextualArticles(r.Context(), route, nil, "", 5)
		if err != nil {
			apierr.WriteInternal(w, r, "Failed to preview route hints.", err)
			return
		}
		writeJSON(w, 200, map[string]any{"articles": articles})
	}
}

func (d Deps) handleMarketingSearchGaps() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, ok := d.marketingAccess(w, r, marketingView); !ok {
			return
		}
		days := 30
		if v := r.URL.Query().Get("days"); v != "" {
			if n, err := parseSearchGapDays(v); err == nil {
				days = n
			}
		}
		since := time.Now().UTC().AddDate(0, 0, -days)
		items, err := mcrepo.SearchGaps(r.Context(), d.Pool, since)
		if err != nil {
			apierr.WriteInternal(w, r, "Failed to load search gaps.", err)
			return
		}
		writeJSON(w, 200, map[string]any{"items": items, "sinceDays": days})
	}
}

func parseSearchGapDays(v string) (int, error) {
	n, err := strconv.Atoi(v)
	if err != nil || n < 1 || n > 365 {
		return 0, strconv.ErrSyntax
	}
	return n, nil
}
