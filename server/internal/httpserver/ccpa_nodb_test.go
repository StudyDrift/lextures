package httpserver

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/lextures/lextures/server/internal/config"
)

func TestCCPA_FeatureDisabled_Returns404(t *testing.T) {
	d := Deps{Config: config.Config{CCPAModuleEnabled: false}}
	h := NewHandler(d)

	paths := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/api/v1/compliance/ccpa/opt-out"},
		{http.MethodPost, "/api/v1/compliance/ccpa/opt-out"},
		{http.MethodPost, "/api/v1/compliance/ccpa/requests"},
		{http.MethodGet, "/api/v1/compliance/ccpa/requests"},
		{http.MethodGet, "/api/v1/compliance/ccpa/requests/00000000-0000-0000-0000-000000000001"},
		{http.MethodPatch, "/api/v1/compliance/ccpa/requests/00000000-0000-0000-0000-000000000001"},
		{http.MethodGet, "/api/v1/compliance/ccpa/pi-categories"},
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

func TestCCPA_Unauthenticated_Returns401(t *testing.T) {
	d := Deps{Config: config.Config{CCPAModuleEnabled: true}}
	h := NewHandler(d)

	paths := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/api/v1/compliance/ccpa/opt-out"},
		{http.MethodPost, "/api/v1/compliance/ccpa/opt-out"},
		{http.MethodPost, "/api/v1/compliance/ccpa/requests"},
		{http.MethodGet, "/api/v1/compliance/ccpa/requests"},
		{http.MethodGet, "/api/v1/compliance/ccpa/requests/00000000-0000-0000-0000-000000000001"},
		{http.MethodPatch, "/api/v1/compliance/ccpa/requests/00000000-0000-0000-0000-000000000001"},
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

func TestCCPA_PICategories_NoAuthRequired(t *testing.T) {
	d := Deps{Config: config.Config{CCPAModuleEnabled: true}}
	h := NewHandler(d)
	r := httptest.NewRequest(http.MethodGet, "/api/v1/compliance/ccpa/pi-categories", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	// No auth → should return 200 (public endpoint).
	if w.Code != http.StatusOK {
		t.Errorf("GET /pi-categories: status=%d want 200", w.Code)
	}
}

func TestCCPA_PostRequest_InvalidJSON_Returns401WhenUnauthenticated(t *testing.T) {
	d := Deps{Config: config.Config{CCPAModuleEnabled: true}}
	h := NewHandler(d)
	r := httptest.NewRequest(http.MethodPost, "/api/v1/compliance/ccpa/requests",
		strings.NewReader("{bad json}"))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	// Auth check fires before JSON parsing → 401.
	if w.Code != http.StatusUnauthorized {
		t.Errorf("status=%d want 401", w.Code)
	}
}

func TestCCPA_PatchRequest_InvalidID_Returns401WhenUnauthenticated(t *testing.T) {
	d := Deps{Config: config.Config{CCPAModuleEnabled: true}}
	h := NewHandler(d)
	r := httptest.NewRequest(http.MethodPatch, "/api/v1/compliance/ccpa/requests/not-a-uuid",
		strings.NewReader(`{"status":"approved"}`))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("status=%d want 401", w.Code)
	}
}

func TestCCPA_GetRequest_InvalidID_Returns401WhenUnauthenticated(t *testing.T) {
	d := Deps{Config: config.Config{CCPAModuleEnabled: true}}
	h := NewHandler(d)
	r := httptest.NewRequest(http.MethodGet, "/api/v1/compliance/ccpa/requests/not-a-uuid", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("status=%d want 401", w.Code)
	}
}

func TestCCPA_GPCHeader_NotAuthenticated_Returns401(t *testing.T) {
	d := Deps{Config: config.Config{CCPAModuleEnabled: true}}
	h := NewHandler(d)
	r := httptest.NewRequest(http.MethodPost, "/api/v1/compliance/ccpa/opt-out", nil)
	r.Header.Set("Sec-GPC", "1")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	// GPC is honoured only for authenticated users → must authenticate first.
	if w.Code != http.StatusUnauthorized {
		t.Errorf("GPC unauthenticated: status=%d want 401", w.Code)
	}
}
