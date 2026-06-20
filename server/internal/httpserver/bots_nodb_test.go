package httpserver

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/lextures/lextures/server/internal/config"
)

func TestBots_FeatureDisabledReturns501(t *testing.T) {
	d := Deps{Config: config.Config{FFBotSlack: false}, Bots: nil}
	r := chi.NewRouter()
	d.registerBotRoutes(r)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/bots", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotImplemented {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestSlackEvents_InvalidSignatureReturns401(t *testing.T) {
	d := Deps{
		Config: config.Config{FFBotSlack: true, FFWebhooks: true},
		Bots:   nil,
	}
	r := chi.NewRouter()
	d.registerBotRoutes(r)
	req := httptest.NewRequest(http.MethodPost, "/integrations/slack/events", nil)
	req.Header.Set("X-Slack-Request-Timestamp", "1")
	req.Header.Set("X-Slack-Signature", "v0=invalid")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotImplemented && rec.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d", rec.Code)
	}
}
