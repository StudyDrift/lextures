package httpserver

import (
	"encoding/json"
	"net/http"
	"strings"
	"github.com/lextures/lextures/server/internal/apierr"
	"github.com/lextures/lextures/server/internal/repos/coursemoduleexternallinks"
	"github.com/lextures/lextures/server/internal/repos/coursestructure"
	"github.com/lextures/lextures/server/internal/service/oersearch"
)

func (d Deps) oerService() *oersearch.Service {
	return oersearch.New(d.Pool, d.effectiveConfig().OERStub)
}

func (d Deps) guardOERLibrary(w http.ResponseWriter) bool {
	if !d.effectiveConfig().OERLibraryEnabled {
		apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "OER library is not enabled.")
		return false
	}
	if d.Pool == nil {
		apierr.WriteJSON(w, http.StatusNotImplemented, apierr.CodeNotImplemented, "Database not configured.")
		return false
	}
	return true
}

// handleGetOERProviders is GET /api/v1/oer/providers.
func (d Deps) handleGetOERProviders() http.HandlerFunc {
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
		if !d.guardOERLibrary(w) {
			return
		}
		ids, err := d.oerService().EnabledProviderIDs(r.Context())
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load OER providers.")
			return
		}
		type row struct {
			Provider string `json:"provider"`
		}
		out := make([]row, 0, len(ids))
		for _, id := range ids {
			out = append(out, row{Provider: id})
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(out)
	}
}

// handleGetOERSearch is GET /api/v1/oer/search.
func (d Deps) handleGetOERSearch() http.HandlerFunc {
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
		if !d.guardOERLibrary(w) {
			return
		}
		provider := strings.TrimSpace(r.URL.Query().Get("provider"))
		if provider == "" {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "provider query parameter is required.")
			return
		}
		params := oersearch.SearchParams{
			Query:   strings.TrimSpace(r.URL.Query().Get("q")),
			Subject: strings.TrimSpace(r.URL.Query().Get("subject")),
			Level:   strings.TrimSpace(r.URL.Query().Get("level")),
			License: strings.TrimSpace(r.URL.Query().Get("license")),
		}
		resp, err := d.oerService().Search(r.Context(), provider, params)
		if err != nil {
			if strings.Contains(err.Error(), "disabled") {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "This OER provider is disabled.")
				return
			}
			apierr.WriteJSON(w, http.StatusBadGateway, apierr.CodeInternal, "OER search is temporarily unavailable.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(resp)
	}
}

type oerImportBody struct {
	Title           string  `json:"title"`
	URL             string  `json:"url"`
	Provider        string  `json:"provider"`
	ExternalID      *string `json:"externalId"`
	LicenseSPDX     *string `json:"licenseSpdx"`
	AttributionText *string `json:"attributionText"`
	ImportCopy      bool    `json:"importCopy"`
}

// handlePostModuleOERImport is POST .../oer-import.
func (d Deps) handlePostModuleOERImport() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost+","+http.MethodOptions)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		if !d.guardOERLibrary(w) {
			return
		}
		_, _, cid, moduleID, ok := d.beginCreateUnderModule(w, r)
		if !ok {
			return
		}
		var body oerImportBody
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		cleanURL, err := coursemoduleexternallinks.ValidateExternalHTTPURL(strings.TrimSpace(body.URL))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, err.Error())
			return
		}
		provider := strings.TrimSpace(body.Provider)
		if provider == "" {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "provider is required.")
			return
		}
		if body.ImportCopy {
			if body.LicenseSPDX != nil && !oersearch.AllowsImportCopy(*body.LicenseSPDX) {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "This license does not allow importing a copy.")
				return
			}
			apierr.WriteJSON(w, http.StatusNotImplemented, apierr.CodeNotImplemented, "Import copy is not available yet; link only.")
			return
		}
		title := strings.TrimSpace(body.Title)
		if title == "" {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "title is required.")
			return
		}
		oerProv := provider
		row, err := coursestructure.InsertExternalLinkUnderModuleWithMeta(
			r.Context(), d.Pool, cid, moduleID, title, cleanURL, provider,
			body.ExternalID, body.LicenseSPDX, body.AttributionText, &oerProv,
		)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to add OER resource.")
			return
		}
		d.writeCreatedStructureItem(w, r, cid, row)
	}
}
