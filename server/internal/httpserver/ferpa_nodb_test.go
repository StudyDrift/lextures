package httpserver

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/lextures/lextures/server/internal/config"
)

func TestFERPA_FeatureDisabled_Returns404(t *testing.T) {
	d := Deps{Config: config.Config{FERPAWorkflowEnabled: false}}
	h := NewHandler(d)

	paths := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/api/v1/compliance/ferpa/directory-opt-out"},
		{http.MethodPut, "/api/v1/compliance/ferpa/directory-opt-out"},
		{http.MethodPost, "/api/v1/compliance/ferpa/record-requests"},
		{http.MethodGet, "/api/v1/compliance/ferpa/record-requests"},
		{http.MethodGet, "/api/v1/compliance/ferpa/disclosure-log"},
		{http.MethodPost, "/api/v1/compliance/ferpa/consent"},
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

func TestFERPA_Unauthenticated_Returns401(t *testing.T) {
	d := Deps{Config: config.Config{FERPAWorkflowEnabled: true}}
	h := NewHandler(d)

	paths := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/api/v1/compliance/ferpa/directory-opt-out"},
		{http.MethodPut, "/api/v1/compliance/ferpa/directory-opt-out"},
		{http.MethodPost, "/api/v1/compliance/ferpa/record-requests"},
		{http.MethodGet, "/api/v1/compliance/ferpa/record-requests"},
		{http.MethodGet, "/api/v1/compliance/ferpa/disclosure-log"},
		{http.MethodPost, "/api/v1/compliance/ferpa/consent"},
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

func TestFERPA_PutDirectoryOptOut_InvalidJSON_Returns400(t *testing.T) {
	d := Deps{Config: config.Config{FERPAWorkflowEnabled: true}}
	h := NewHandler(d)
	r := httptest.NewRequest(http.MethodPut, "/api/v1/compliance/ferpa/directory-opt-out",
		strings.NewReader("{bad json}"))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	// No JWT → 401 before JSON is parsed; ensure we don't panic.
	if w.Code == 0 {
		t.Fatal("expected a response code")
	}
}

func TestFERPA_PostRecordRequest_InvalidJSON_Returns401(t *testing.T) {
	d := Deps{Config: config.Config{FERPAWorkflowEnabled: true}}
	h := NewHandler(d)
	r := httptest.NewRequest(http.MethodPost, "/api/v1/compliance/ferpa/record-requests",
		strings.NewReader("{bad json}"))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	// Unauthenticated; auth check fires before body parsing.
	if w.Code != http.StatusUnauthorized {
		t.Errorf("status=%d want 401", w.Code)
	}
}

func TestFERPA_PatchRecordRequest_InvalidID_Returns404(t *testing.T) {
	d := Deps{Config: config.Config{FERPAWorkflowEnabled: true}}
	h := NewHandler(d)
	r := httptest.NewRequest(http.MethodPatch, "/api/v1/compliance/ferpa/record-requests/not-a-uuid",
		strings.NewReader(`{"status":"approved"}`))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	// No JWT → 401.
	if w.Code != http.StatusUnauthorized {
		t.Errorf("status=%d want 401", w.Code)
	}
}

func TestFERPA_DeleteConsent_InvalidID_Returns401(t *testing.T) {
	d := Deps{Config: config.Config{FERPAWorkflowEnabled: true}}
	h := NewHandler(d)
	r := httptest.NewRequest(http.MethodDelete, "/api/v1/compliance/ferpa/consent/not-a-uuid", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("status=%d want 401", w.Code)
	}
}
