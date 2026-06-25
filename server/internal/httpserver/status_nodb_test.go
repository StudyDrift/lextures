package httpserver

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/lextures/lextures/server/internal/config"
	statuspageservice "github.com/lextures/lextures/server/internal/service/statuspage"
)

func TestStatusSummary_Public_ReturnsOperationalWhenDisabled(t *testing.T) {
	d := Deps{Config: config.Config{StatusPageEnabled: false}}
	h := NewHandler(d)

	r := httptest.NewRequest(http.MethodGet, "/api/v1/status-summary", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	if !bytes.Contains(w.Body.Bytes(), []byte(`"incidents"`)) {
		t.Fatalf("body=%s", w.Body.String())
	}
}

func TestAlertmanagerWebhook_Disabled_Returns404(t *testing.T) {
	d := Deps{Config: config.Config{StatusPageEnabled: false}}
	h := NewHandler(d)
	r := httptest.NewRequest(http.MethodPost, "/api/v1/internal/ops/alertmanager-webhook", bytes.NewReader([]byte(`{"alerts":[]}`)))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusNotFound {
		t.Fatalf("status=%d", w.Code)
	}
}

func TestAlertmanagerWebhook_InvalidAuth_Returns401(t *testing.T) {
	d := Deps{Config: config.Config{
		StatusPageEnabled:         true,
		AlertmanagerWebhookSecret: "secret-token",
	}}
	h := NewHandler(d)
	r := httptest.NewRequest(http.MethodPost, "/api/v1/internal/ops/alertmanager-webhook", bytes.NewReader([]byte(`{"alerts":[]}`)))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d", w.Code)
	}
}

func TestAlertmanagerWebhook_UpdatesComponent(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Fatalf("method=%s", r.Method)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer upstream.Close()

	client := statuspageservice.NewClient(statuspageservice.Config{
		Enabled:      true,
		PageURL:      "https://status.lextures.io",
		APIKey:       "key",
		PageID:       "page-1",
		APIBaseURL:   upstream.URL,
		HTTPClient:   upstream.Client(),
		CacheTTL:     time.Minute,
		ComponentMap: statuspageservice.ComponentMap{"api": "comp-api"},
	})

	d := Deps{
		Config: config.Config{
			StatusPageEnabled:         true,
			AlertmanagerWebhookSecret: "secret-token",
		},
		StatusPageClient: client,
	}
	h := NewHandler(d)

	body := []byte(`{
		"status":"firing",
		"alerts":[{"status":"firing","labels":{"statuspage_component":"api","severity":"warning"}}]
	}`)
	r := httptest.NewRequest(http.MethodPost, "/api/v1/internal/ops/alertmanager-webhook", bytes.NewReader(body))
	r.Header.Set("Authorization", "Bearer secret-token")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusNoContent {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
}