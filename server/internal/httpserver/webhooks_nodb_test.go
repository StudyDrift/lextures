package httpserver

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/lextures/lextures/server/internal/config"
)

func TestWebhooks_FeatureDisabledReturns501(t *testing.T) {
	d := Deps{Config: config.Config{FFWebhooks: false}}
	r := chi.NewRouter()
	d.registerWebhookRoutes(r)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/webhooks", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotImplemented {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestWebhooks_EventTypesPublicWhenEnabled(t *testing.T) {
	d := Deps{Config: config.Config{FFWebhooks: true}}
	r := chi.NewRouter()
	d.registerWebhookRoutes(r)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/webhooks/event-types", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "grade.posted") {
		t.Fatalf("body=%s", rec.Body.String())
	}
}

func TestWebhooks_ListUnauthenticatedReturns401(t *testing.T) {
	d := Deps{Config: config.Config{FFWebhooks: true}}
	r := chi.NewRouter()
	d.registerWebhookRoutes(r)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/webhooks", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d", rec.Code)
	}
}
