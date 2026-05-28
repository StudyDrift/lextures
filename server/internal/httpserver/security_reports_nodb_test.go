package httpserver

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/lextures/lextures/server/internal/config"
)

func TestSecurityReports_TrustEndpoint_Public(t *testing.T) {
	d := Deps{Config: config.Config{SecurityDisclosureModuleEnabled: false}}
	h := NewHandler(d)
	r := httptest.NewRequest(http.MethodGet, "/api/v1/trust/security", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("trust/security: status=%d want 200", w.Code)
	}
	if !strings.Contains(w.Body.String(), "security@lextures.io") {
		t.Error("expected contact email in response")
	}
}

func TestSecurityReports_Admin_FeatureDisabled_Returns404(t *testing.T) {
	d := Deps{Config: config.Config{SecurityDisclosureModuleEnabled: false}}
	h := NewHandler(d)
	paths := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/api/v1/compliance/security-reports"},
		{http.MethodPost, "/api/v1/compliance/security-reports"},
		{http.MethodGet, "/api/v1/compliance/security-reports/export"},
		{http.MethodGet, "/api/v1/compliance/security-reports/00000000-0000-0000-0000-000000000001"},
		{http.MethodPatch, "/api/v1/compliance/security-reports/00000000-0000-0000-0000-000000000001"},
	}
	for _, p := range paths {
		r := httptest.NewRequest(p.method, p.path, nil)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != http.StatusNotFound {
			t.Errorf("%s %s: status=%d want 404", p.method, p.path, w.Code)
		}
	}
}

func TestSecurityReports_Admin_Unauthenticated_Returns401(t *testing.T) {
	d := Deps{Config: config.Config{SecurityDisclosureModuleEnabled: true}}
	h := NewHandler(d)
	r := httptest.NewRequest(http.MethodGet, "/api/v1/compliance/security-reports", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d want 401", w.Code)
	}
}
