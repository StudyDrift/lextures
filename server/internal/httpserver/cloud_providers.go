package httpserver

import (
	"encoding/json"
	"net/http"

	"github.com/lextures/lextures/server/internal/apierr"
	"github.com/lextures/lextures/server/internal/repos/cloudproviders"
)

type cloudProviderPublicRow struct {
	Provider string `json:"provider"`
	ClientID string `json:"clientId,omitempty"`
	APIKey   string `json:"apiKey,omitempty"`
	AppKey   string `json:"appKey,omitempty"`
}

// handleGetCloudProviders is GET /api/v1/cloud-providers (authenticated; enabled + configured only).
func (d Deps) handleGetCloudProviders() http.HandlerFunc {
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
		if _, ok := d.meUserID(w, r); !ok {
			return
		}
		if d.Pool == nil {
			apierr.WriteJSON(w, http.StatusNotImplemented, apierr.CodeNotImplemented, "Database not configured.")
			return
		}
		list, err := cloudproviders.EnabledConfigured(r.Context(), d.Pool)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load cloud providers.")
			return
		}
		out := make([]cloudProviderPublicRow, 0, len(list))
		for _, p := range list {
			row := cloudProviderPublicRow{Provider: p.Provider}
			switch p.Provider {
			case "google_drive":
				row.ClientID = p.ClientID
				row.APIKey = p.APIKey
			case "onedrive":
				row.ClientID = p.ClientID
			case "dropbox":
				row.AppKey = p.AppKey
			}
			out = append(out, row)
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(out)
	}
}
