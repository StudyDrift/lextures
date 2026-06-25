package httpserver

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/lextures/lextures/server/internal/apierr"
	statuspageservice "github.com/lextures/lextures/server/internal/service/statuspage"
)

func (d Deps) statusPageClient() *statuspageservice.Client {
	if d.StatusPageClient != nil {
		return d.StatusPageClient
	}
	cfg := d.effectiveConfig()
	componentMap, err := statuspageservice.ParseComponentMap(cfg.StatuspageComponentMapJSON)
	if err != nil {
		componentMap = statuspageservice.ComponentMap{}
	}
	return statuspageservice.NewClient(statuspageservice.Config{
		Enabled:      cfg.StatusPageEnabled,
		PageURL:      cfg.StatusPageURL,
		APIKey:       cfg.StatuspageAPIKey,
		PageID:       cfg.StatuspagePageID,
		ComponentMap: componentMap,
		CacheTTL:     time.Duration(cfg.StatusPageSummaryCacheSecs) * time.Second,
	})
}

func (d Deps) registerStatusRoutes(r chi.Router) {
	r.Get("/api/v1/status-summary", d.handleGetStatusSummary())
	r.Post("/api/v1/internal/ops/alertmanager-webhook", d.handleAlertmanagerWebhook())
}

// GET /api/v1/status-summary
func (d Deps) handleGetStatusSummary() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		summary, err := d.statusPageClient().Summary(r.Context())
		if err != nil && len(summary.Incidents) == 0 {
			apierr.WriteJSON(w, http.StatusBadGateway, apierr.CodeInternal, "Could not load status summary.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Header().Set("Cache-Control", "public, max-age=60")
		_ = json.NewEncoder(w).Encode(summary)
	}
}

// POST /api/v1/internal/ops/alertmanager-webhook
func (d Deps) handleAlertmanagerWebhook() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cfg := d.effectiveConfig()
		if !cfg.StatusPageEnabled {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Status page integration is not enabled.")
			return
		}
		if !d.verifyAlertmanagerWebhookAuth(r, cfg.AlertmanagerWebhookSecret) {
			apierr.WriteJSON(w, http.StatusUnauthorized, apierr.CodeUnauthorized, "Invalid webhook credentials.")
			return
		}
		body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Could not read request body.")
			return
		}
		payload, err := statuspageservice.ParseAlertmanagerWebhook(body)
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid Alertmanager payload.")
			return
		}
		if err := d.statusPageClient().ApplyAlertmanagerWebhook(r.Context(), payload); err != nil {
			if strings.Contains(err.Error(), "not configured") {
				apierr.WriteJSON(w, http.StatusServiceUnavailable, apierr.CodeInternal, "Statuspage is not configured.")
				return
			}
			apierr.WriteJSON(w, http.StatusBadGateway, apierr.CodeInternal, "Could not update status page components.")
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func (d Deps) verifyAlertmanagerWebhookAuth(r *http.Request, secret string) bool {
	secret = strings.TrimSpace(secret)
	if secret == "" {
		return false
	}
	auth := strings.TrimSpace(r.Header.Get("Authorization"))
	if strings.HasPrefix(strings.ToLower(auth), "bearer ") {
		return subtleConstantTimeEqual(auth[7:], secret)
	}
	if token := strings.TrimSpace(r.Header.Get("X-Alertmanager-Token")); token != "" {
		return subtleConstantTimeEqual(token, secret)
	}
	return false
}

func subtleConstantTimeEqual(a, b string) bool {
	a = strings.TrimSpace(a)
	b = strings.TrimSpace(b)
	if len(a) != len(b) {
		return false
	}
	var diff byte
	for i := 0; i < len(a); i++ {
		diff |= a[i] ^ b[i]
	}
	return diff == 0
}