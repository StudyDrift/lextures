package httpserver

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/lextures/lextures/server/internal/apierr"
	"github.com/lextures/lextures/server/internal/repos/oerproviders"
)

var validOERProviders = map[string]bool{
	"oer_commons": true,
	"merlot":      true,
	"openstax":    true,
}

// handleGetAdminOERProviders is GET /api/v1/admin/oer-providers.
func (d Deps) handleGetAdminOERProviders() http.HandlerFunc {
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
		if _, ok := d.adminRbacUser(w, r); !ok {
			return
		}
		if !d.guardOERLibrary(w) {
			return
		}
		list, err := oerproviders.List(r.Context(), d.Pool)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load OER provider settings.")
			return
		}
		type row struct {
			Provider  string `json:"provider"`
			Enabled   bool   `json:"enabled"`
			UpdatedAt string `json:"updatedAt"`
		}
		out := make([]row, 0, len(list))
		for _, p := range list {
			out = append(out, row{
				Provider:  p.Provider,
				Enabled:   p.Enabled,
				UpdatedAt: p.UpdatedAt.UTC().Format("2006-01-02T15:04:05Z"),
			})
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(out)
	}
}

// handlePutAdminOERProvider is PUT /api/v1/admin/oer-providers/{provider}.
func (d Deps) handlePutAdminOERProvider() http.HandlerFunc {
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
		if _, ok := d.adminRbacUser(w, r); !ok {
			return
		}
		if !d.guardOERLibrary(w) {
			return
		}
		provider := strings.ToLower(strings.TrimSpace(chi.URLParam(r, "provider")))
		if !validOERProviders[provider] {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Unknown provider. Must be oer_commons, merlot, or openstax.")
			return
		}
		var body struct {
			Enabled bool `json:"enabled"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		if err := oerproviders.SetEnabled(r.Context(), d.Pool, provider, body.Enabled); err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to update OER provider setting.")
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}
