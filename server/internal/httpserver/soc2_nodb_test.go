package httpserver

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/lextures/lextures/server/internal/config"
)

func TestSOC2_FeatureDisabled_Returns404(t *testing.T) {
	d := Deps{Config: config.Config{SOC2ModuleEnabled: false}}
	h := NewHandler(d)

	paths := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/api/v1/internal/compliance/soc2/evidence-summary"},
		{http.MethodGet, "/api/v1/internal/compliance/soc2/access-reviews"},
		{http.MethodPost, "/api/v1/internal/compliance/soc2/access-reviews"},
		{http.MethodPost, "/api/v1/internal/compliance/soc2/incidents"},
		{http.MethodGet, "/api/v1/internal/compliance/soc2/incidents"},
		{http.MethodGet, "/api/v1/internal/compliance/soc2/incidents/00000000-0000-0000-0000-000000000001"},
		{http.MethodPatch, "/api/v1/internal/compliance/soc2/incidents/00000000-0000-0000-0000-000000000001"},
		{http.MethodGet, "/api/v1/internal/compliance/soc2/vendor-risk"},
		{http.MethodPost, "/api/v1/internal/compliance/soc2/vendor-risk"},
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

func TestSOC2_Unauthenticated_Returns401(t *testing.T) {
	d := Deps{Config: config.Config{SOC2ModuleEnabled: true}}
	h := NewHandler(d)

	paths := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/api/v1/internal/compliance/soc2/evidence-summary"},
		{http.MethodGet, "/api/v1/internal/compliance/soc2/access-reviews"},
		{http.MethodPost, "/api/v1/internal/compliance/soc2/access-reviews"},
		{http.MethodPost, "/api/v1/internal/compliance/soc2/incidents"},
		{http.MethodGet, "/api/v1/internal/compliance/soc2/incidents"},
		{http.MethodGet, "/api/v1/internal/compliance/soc2/incidents/00000000-0000-0000-0000-000000000001"},
		{http.MethodPatch, "/api/v1/internal/compliance/soc2/incidents/00000000-0000-0000-0000-000000000001"},
		{http.MethodGet, "/api/v1/internal/compliance/soc2/vendor-risk"},
		{http.MethodPost, "/api/v1/internal/compliance/soc2/vendor-risk"},
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

func TestSOC2_PostIncident_InvalidJSON_Returns401WhenUnauthenticated(t *testing.T) {
	d := Deps{Config: config.Config{SOC2ModuleEnabled: true}}
	h := NewHandler(d)
	r := httptest.NewRequest(http.MethodPost, "/api/v1/internal/compliance/soc2/incidents",
		strings.NewReader("{bad json}"))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	// Auth check fires before JSON parsing.
	if w.Code != http.StatusUnauthorized {
		t.Errorf("status=%d want 401", w.Code)
	}
}

func TestSOC2_GetIncident_InvalidID_Returns401WhenUnauthenticated(t *testing.T) {
	d := Deps{Config: config.Config{SOC2ModuleEnabled: true}}
	h := NewHandler(d)
	r := httptest.NewRequest(http.MethodGet, "/api/v1/internal/compliance/soc2/incidents/not-a-uuid", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("status=%d want 401", w.Code)
	}
}

func TestSOC2_PatchIncident_InvalidID_Returns401WhenUnauthenticated(t *testing.T) {
	d := Deps{Config: config.Config{SOC2ModuleEnabled: true}}
	h := NewHandler(d)
	r := httptest.NewRequest(http.MethodPatch, "/api/v1/internal/compliance/soc2/incidents/not-a-uuid",
		strings.NewReader(`{"status":"resolved"}`))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("status=%d want 401", w.Code)
	}
}
