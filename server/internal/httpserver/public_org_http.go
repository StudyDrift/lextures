package httpserver

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/lextures/lextures/server/internal/apierr"
	"github.com/lextures/lextures/server/internal/repos/organization"
)

// GET /api/v1/public/orgs/by-slug/{slug}
func (d Deps) handlePublicOrgBySlug() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		if d.Pool == nil {
			apierr.WriteJSON(w, http.StatusServiceUnavailable, apierr.CodeInvalidInput, "Database is not configured.")
			return
		}
		slug := organization.NormalizeSlug(chi.URLParam(r, "slug"))
		if slug == "" {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Organization slug is required.")
			return
		}
		row, err := organization.GetBySlug(r.Context(), d.Pool, slug)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load organization.")
			return
		}
		if row == nil || row.Status != "active" {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Organization not found.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]string{
			"slug": row.Slug,
			"name": row.Name,
		})
	}
}

func orgSlugFromBrandingQuery(r *http.Request) string {
	if v := strings.TrimSpace(r.URL.Query().Get("orgSlug")); v != "" {
		return organization.NormalizeSlug(v)
	}
	if v := strings.TrimSpace(r.URL.Query().Get("org_slug")); v != "" {
		return organization.NormalizeSlug(v)
	}
	return ""
}