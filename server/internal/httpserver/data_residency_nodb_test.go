package httpserver

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/lextures/lextures/server/internal/config"
)

func TestDataResidency_FeatureDisabled_Returns404(t *testing.T) {
	d := Deps{Config: config.Config{DataResidencyEnabled: false}}
	h := NewHandler(d)

	paths := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/api/v1/internal/compliance/data-residency/org/00000000-0000-0000-0000-000000000001"},
		{http.MethodGet, "/api/v1/internal/compliance/data-residency/access-log"},
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

func TestDataResidency_Unauthenticated_Returns401(t *testing.T) {
	d := Deps{Config: config.Config{DataResidencyEnabled: true}}
	h := NewHandler(d)

	paths := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/api/v1/internal/compliance/data-residency/org/00000000-0000-0000-0000-000000000001"},
		{http.MethodGet, "/api/v1/internal/compliance/data-residency/access-log"},
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

func TestDataResidency_GetOrg_InvalidID_Returns401WhenUnauthenticated(t *testing.T) {
	d := Deps{Config: config.Config{DataResidencyEnabled: true}}
	h := NewHandler(d)
	r := httptest.NewRequest(http.MethodGet, "/api/v1/internal/compliance/data-residency/org/not-a-uuid", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	// Auth check fires before ID parsing.
	if w.Code != http.StatusUnauthorized {
		t.Errorf("status=%d want 401", w.Code)
	}
}
