package httpserver

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/lextures/lextures/server/internal/config"
)

func TestMaintenanceBanner_FeatureDisabled_ReturnsNull(t *testing.T) {
	d := Deps{Config: config.Config{MaintenanceBannerEnabled: false}}
	h := NewHandler(d)
	r := httptest.NewRequest(http.MethodGet, "/api/v1/status/banner", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d want 200", w.Code)
	}
	if body := w.Body.String(); body != "null\n" && body != "null" {
		t.Fatalf("body=%q want null", body)
	}
}

func TestMaintenanceBanner_AdminDisabled_Returns404(t *testing.T) {
	d := Deps{Config: config.Config{MaintenanceBannerEnabled: false}}
	h := NewHandler(d)
	r := httptest.NewRequest(http.MethodGet, "/api/v1/admin/banners", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusNotFound {
		t.Fatalf("status=%d want 404", w.Code)
	}
}

func TestMaintenanceBanner_AdminUnauthenticated_Returns401(t *testing.T) {
	d := Deps{Config: config.Config{MaintenanceBannerEnabled: true, AdminConsoleEnabled: true}}
	h := NewHandler(d)
	r := httptest.NewRequest(http.MethodPost, "/api/v1/admin/banners", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d want 401", w.Code)
	}
}

func TestMaintenanceBanner_StatuspageWebhook_NoHMAC_Returns401(t *testing.T) {
	d := Deps{Config: config.Config{
		MaintenanceBannerEnabled:  true,
		StatuspageWebhookSecret:   "test-secret",
	}}
	h := NewHandler(d)
	r := httptest.NewRequest(http.MethodPost, "/api/v1/admin/banners/statuspage-webhook", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d want 401", w.Code)
	}
}

func TestVerifyStatuspageHMAC(t *testing.T) {
	secret := "shh"
	body := []byte(`{"incident":{"id":"1","name":"Outage","status":"investigating","impact":"major"}}`)
	mac := httptest.NewRequest(http.MethodPost, "/", nil)
	mac.Header.Set("X-Statuspage-Signature", "deadbeef")
	if verifyStatuspageHMAC(mac, secret, body) {
		t.Fatal("expected invalid signature to fail")
	}
}
