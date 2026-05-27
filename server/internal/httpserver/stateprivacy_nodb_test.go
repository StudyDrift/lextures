package httpserver

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/lextures/lextures/server/internal/config"
)

func TestStatePrivacy_FeatureDisabled_Returns404(t *testing.T) {
	d := Deps{Config: config.Config{StatePrivacyEnabled: false}}
	h := NewHandler(d)

	paths := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/api/v1/compliance/state/disclosure/00000000-0000-0000-0000-000000000001"},
		{http.MethodPost, "/api/v1/compliance/state/deletion-request"},
		{http.MethodGet, "/api/v1/compliance/state/deletion-request/00000000-0000-0000-0000-000000000001"},
		{http.MethodPatch, "/api/v1/compliance/state/deletion-request/00000000-0000-0000-0000-000000000001"},
		{http.MethodGet, "/api/v1/compliance/state/checklist"},
		{http.MethodGet, "/api/v1/compliance/state/dpa-addendum/CA"},
		{http.MethodGet, "/api/v1/compliance/state/prohibitions"},
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

func TestStatePrivacy_Unauthenticated_Returns401(t *testing.T) {
	d := Deps{Config: config.Config{StatePrivacyEnabled: true}}
	h := NewHandler(d)

	paths := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/api/v1/compliance/state/disclosure/00000000-0000-0000-0000-000000000001"},
		{http.MethodPost, "/api/v1/compliance/state/deletion-request"},
		{http.MethodGet, "/api/v1/compliance/state/deletion-request/00000000-0000-0000-0000-000000000001"},
		{http.MethodPatch, "/api/v1/compliance/state/deletion-request/00000000-0000-0000-0000-000000000001"},
		{http.MethodGet, "/api/v1/compliance/state/checklist"},
		{http.MethodGet, "/api/v1/compliance/state/dpa-addendum/CA"},
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

func TestStatePrivacy_Prohibitions_NoAuthRequired(t *testing.T) {
	d := Deps{Config: config.Config{StatePrivacyEnabled: true}}
	h := NewHandler(d)
	r := httptest.NewRequest(http.MethodGet, "/api/v1/compliance/state/prohibitions", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Errorf("GET /prohibitions: status=%d want 200", w.Code)
	}
}

func TestStatePrivacy_PostDeletionRequest_InvalidJSON_Returns401WhenUnauthenticated(t *testing.T) {
	d := Deps{Config: config.Config{StatePrivacyEnabled: true}}
	h := NewHandler(d)
	r := httptest.NewRequest(http.MethodPost, "/api/v1/compliance/state/deletion-request",
		strings.NewReader("{bad json}"))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	// Auth check fires before JSON parsing.
	if w.Code != http.StatusUnauthorized {
		t.Errorf("status=%d want 401", w.Code)
	}
}

func TestStatePrivacy_DPAAddendum_InvalidID_Returns401WhenUnauthenticated(t *testing.T) {
	d := Deps{Config: config.Config{StatePrivacyEnabled: true}}
	h := NewHandler(d)
	r := httptest.NewRequest(http.MethodGet, "/api/v1/compliance/state/dpa-addendum/TX", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	// Auth required before state validation.
	if w.Code != http.StatusUnauthorized {
		t.Errorf("status=%d want 401", w.Code)
	}
}

func TestStatePrivacy_PatchDeletionRequest_InvalidID_Returns401WhenUnauthenticated(t *testing.T) {
	d := Deps{Config: config.Config{StatePrivacyEnabled: true}}
	h := NewHandler(d)
	r := httptest.NewRequest(http.MethodPatch, "/api/v1/compliance/state/deletion-request/not-a-uuid",
		strings.NewReader(`{"status":"completed"}`))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("status=%d want 401", w.Code)
	}
}
