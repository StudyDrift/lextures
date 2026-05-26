package httpserver

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/lextures/lextures/server/internal/config"
)

func TestGDPR_FeatureDisabled_Returns404(t *testing.T) {
	d := Deps{Config: config.Config{GDPRModuleEnabled: false}}
	h := NewHandler(d)

	paths := []struct {
		method string
		path   string
	}{
		{http.MethodPost, "/api/v1/compliance/gdpr/dsar"},
		{http.MethodGet, "/api/v1/compliance/gdpr/dsar"},
		{http.MethodGet, "/api/v1/compliance/gdpr/dsar/00000000-0000-0000-0000-000000000001/download"},
		{http.MethodPatch, "/api/v1/compliance/gdpr/dsar/00000000-0000-0000-0000-000000000001"},
		{http.MethodPost, "/api/v1/compliance/gdpr/consents"},
		{http.MethodGet, "/api/v1/compliance/gdpr/consents"},
		{http.MethodDelete, "/api/v1/compliance/gdpr/consents/00000000-0000-0000-0000-000000000001"},
		{http.MethodGet, "/api/v1/compliance/gdpr/ropa"},
		{http.MethodPost, "/api/v1/compliance/gdpr/ropa"},
		{http.MethodGet, "/api/v1/compliance/gdpr/dpa-template"},
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

func TestGDPR_Unauthenticated_Returns401(t *testing.T) {
	d := Deps{Config: config.Config{GDPRModuleEnabled: true}}
	h := NewHandler(d)

	paths := []struct {
		method string
		path   string
	}{
		{http.MethodPost, "/api/v1/compliance/gdpr/dsar"},
		{http.MethodGet, "/api/v1/compliance/gdpr/dsar"},
		{http.MethodGet, "/api/v1/compliance/gdpr/dsar/00000000-0000-0000-0000-000000000001/download"},
		{http.MethodPost, "/api/v1/compliance/gdpr/consents"},
		{http.MethodGet, "/api/v1/compliance/gdpr/consents"},
		{http.MethodDelete, "/api/v1/compliance/gdpr/consents/00000000-0000-0000-0000-000000000001"},
		{http.MethodGet, "/api/v1/compliance/gdpr/dpa-template"},
	}
	for _, p := range paths {
		r := httptest.NewRequest(p.method, p.path, nil)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != http.StatusUnauthorized {
			t.Errorf("%s %s: status=%d want 401", p.method, p.path, w.Code)
		}
	}
}

func TestGDPR_PostDSAR_InvalidJSON_Returns401WhenUnauthenticated(t *testing.T) {
	d := Deps{Config: config.Config{GDPRModuleEnabled: true}}
	h := NewHandler(d)
	r := httptest.NewRequest(http.MethodPost, "/api/v1/compliance/gdpr/dsar",
		strings.NewReader("{bad json}"))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	// Auth check fires before JSON parsing → 401.
	if w.Code != http.StatusUnauthorized {
		t.Errorf("status=%d want 401", w.Code)
	}
}

func TestGDPR_PatchDSAR_InvalidID_Returns401WhenUnauthenticated(t *testing.T) {
	d := Deps{Config: config.Config{GDPRModuleEnabled: true}}
	h := NewHandler(d)
	r := httptest.NewRequest(http.MethodPatch, "/api/v1/compliance/gdpr/dsar/not-a-uuid",
		strings.NewReader(`{"status":"approved"}`))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("status=%d want 401", w.Code)
	}
}

func TestGDPR_DeleteConsent_InvalidID_Returns401WhenUnauthenticated(t *testing.T) {
	d := Deps{Config: config.Config{GDPRModuleEnabled: true}}
	h := NewHandler(d)
	r := httptest.NewRequest(http.MethodDelete, "/api/v1/compliance/gdpr/consents/not-a-uuid", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("status=%d want 401", w.Code)
	}
}

func TestGDPR_DSARDownload_InvalidID_Returns401WhenUnauthenticated(t *testing.T) {
	d := Deps{Config: config.Config{GDPRModuleEnabled: true}}
	h := NewHandler(d)
	r := httptest.NewRequest(http.MethodGet, "/api/v1/compliance/gdpr/dsar/not-a-uuid/download", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("status=%d want 401", w.Code)
	}
}

func TestGDPRIPHash_Deterministic(t *testing.T) {
	h1 := gdprIPHash("192.168.1.1:12345")
	h2 := gdprIPHash("192.168.1.1:12345")
	if h1 != h2 {
		t.Error("gdprIPHash should be deterministic")
	}
	if len(h1) != 64 {
		t.Errorf("gdprIPHash len=%d want 64 (SHA-256 hex)", len(h1))
	}
}

func TestGDPRIPHash_DifferentIPs(t *testing.T) {
	h1 := gdprIPHash("192.168.1.1:12345")
	h2 := gdprIPHash("10.0.0.1:12345")
	if h1 == h2 {
		t.Error("gdprIPHash should differ for different IPs")
	}
}

func TestGDPRNonNilSlice_NilInput(t *testing.T) {
	s := gdprNonNilSlice(nil)
	if s == nil {
		t.Error("gdprNonNilSlice(nil) should return non-nil empty slice")
	}
	if len(s) != 0 {
		t.Errorf("gdprNonNilSlice(nil) len=%d want 0", len(s))
	}
}
