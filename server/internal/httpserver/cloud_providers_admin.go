package httpserver

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/lextures/lextures/server/internal/apierr"
	"github.com/lextures/lextures/server/internal/repos/cloudproviders"
)

var validCloudProviders = map[string]bool{
	"google_drive": true,
	"onedrive":     true,
	"dropbox":      true,
}

// handleGetAdminCloudProviders is GET /api/v1/admin/cloud-providers.
func (d Deps) handleGetAdminCloudProviders() http.HandlerFunc {
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
		list, err := cloudproviders.List(r.Context(), d.Pool)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load cloud provider settings.")
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

// handlePutAdminCloudProvider is PUT /api/v1/admin/cloud-providers/{provider}.
func (d Deps) handlePutAdminCloudProvider() http.HandlerFunc {
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
		provider := strings.ToLower(strings.TrimSpace(chi.URLParam(r, "provider")))
		if !validCloudProviders[provider] {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Unknown provider. Must be google_drive, onedrive, or dropbox.")
			return
		}
		var body struct {
			Enabled bool `json:"enabled"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		if err := cloudproviders.SetEnabled(r.Context(), d.Pool, provider, body.Enabled); err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to update cloud provider setting.")
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}
