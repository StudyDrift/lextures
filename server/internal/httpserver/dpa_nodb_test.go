package httpserver

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/lextures/lextures/server/internal/config"
)

func TestDPA_FeatureDisabled_Returns404(t *testing.T) {
	d := Deps{Config: config.Config{DPAPortalEnabled: false}}
	h := NewHandler(d)

	paths := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/api/v1/compliance/dpa/current"},
		{http.MethodPost, "/api/v1/compliance/dpa/accept"},
		{http.MethodGet, "/api/v1/compliance/dpa/acceptances"},
		{http.MethodGet, "/api/v1/compliance/data-inventory"},
		{http.MethodGet, "/api/v1/compliance/data-inventory/export.csv"},
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

func TestDPA_Unauthenticated_Returns401(t *testing.T) {
	d := Deps{Config: config.Config{DPAPortalEnabled: true}}
	h := NewHandler(d)

	paths := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/api/v1/compliance/dpa/current"},
		{http.MethodPost, "/api/v1/compliance/dpa/accept"},
		{http.MethodGet, "/api/v1/compliance/dpa/acceptances"},
		{http.MethodGet, "/api/v1/compliance/data-inventory"},
		{http.MethodGet, "/api/v1/compliance/data-inventory/export.csv"},
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

func TestDPASlice_NilInput(t *testing.T) {
	s := dpaSlice(nil)
	if s == nil {
		t.Error("dpaSlice(nil) should return non-nil empty slice")
	}
	if len(s) != 0 {
		t.Errorf("dpaSlice(nil) len=%d want 0", len(s))
	}
}

func TestDPASlice_NonNilPassthrough(t *testing.T) {
	in := []string{"a", "b"}
	out := dpaSlice(in)
	if len(out) != 2 {
		t.Errorf("dpaSlice len=%d want 2", len(out))
	}
}
