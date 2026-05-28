package httpserver

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/lextures/lextures/server/internal/config"
)

func TestISO_FeatureDisabled_Returns404(t *testing.T) {
	d := Deps{Config: config.Config{IsoIsmsEnabled: false}}
	h := NewHandler(d)

	paths := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/api/v1/compliance/iso/dashboard"},
		{http.MethodGet, "/api/v1/compliance/iso/audit-findings"},
		{http.MethodPost, "/api/v1/compliance/iso/audit-findings"},
		{http.MethodPatch, "/api/v1/compliance/iso/audit-findings/00000000-0000-0000-0000-000000000001"},
		{http.MethodGet, "/api/v1/compliance/iso/risk-register"},
		{http.MethodPost, "/api/v1/compliance/iso/risk-register"},
		{http.MethodGet, "/api/v1/compliance/iso/supplier-reviews"},
		{http.MethodPost, "/api/v1/compliance/iso/supplier-reviews"},
		{http.MethodGet, "/api/v1/compliance/iso/training"},
		{http.MethodPost, "/api/v1/compliance/iso/training"},
		{http.MethodGet, "/api/v1/compliance/iso/soa"},
		{http.MethodPatch, "/api/v1/compliance/iso/soa/A.8.2"},
		{http.MethodPatch, "/api/v1/compliance/iso/program"},
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

func TestISO_Unauthenticated_Returns401(t *testing.T) {
	d := Deps{Config: config.Config{IsoIsmsEnabled: true}}
	h := NewHandler(d)

	paths := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/api/v1/compliance/iso/dashboard"},
		{http.MethodGet, "/api/v1/compliance/iso/audit-findings"},
		{http.MethodPost, "/api/v1/compliance/iso/audit-findings"},
		{http.MethodGet, "/api/v1/compliance/iso/risk-register"},
		{http.MethodGet, "/api/v1/compliance/iso/soa"},
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

func TestISO_TrustEndpoint_NoAuth_NoPool_Returns503(t *testing.T) {
	d := Deps{Config: config.Config{IsoIsmsEnabled: true}}
	h := NewHandler(d)
	r := httptest.NewRequest(http.MethodGet, "/api/v1/trust/iso", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("status=%d want 503", w.Code)
	}
}

func TestISO_PostAuditFinding_Unauthenticated_Returns401(t *testing.T) {
	d := Deps{Config: config.Config{IsoIsmsEnabled: true}}
	h := NewHandler(d)
	r := httptest.NewRequest(http.MethodPost, "/api/v1/compliance/iso/audit-findings",
		strings.NewReader(`{"auditCycle":"2026-internal","findingType":"observation","isoClause":"A.8.15","description":"test"}`))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("status=%d want 401", w.Code)
	}
}
