package httpserver

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/lextures/lextures/server/internal/auth"
)

func TestInsights_Unauthorized(t *testing.T) {
	h := NewHandler(Deps{Pool: nil, JWTSigner: nil})
	rr := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api/v1/courses/cs101/analytics/insights", nil)
	h.ServeHTTP(rr, r)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("want 401, got %d", rr.Code)
	}
}

func TestInsights_MethodNotAllowed(t *testing.T) {
	s := auth.NewJWTSigner("01234567890123456789012345678901")
	h := NewHandler(Deps{Pool: nil, JWTSigner: s})
	rr := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPut, "/api/v1/courses/cs101/analytics/insights", nil)
	r.Header.Set("Authorization", "Bearer x")
	h.ServeHTTP(rr, r)
	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("want 405, got %d", rr.Code)
	}
}

func TestInsights_RouteRegistered(t *testing.T) {
	s := auth.NewJWTSigner("01234567890123456789012345678901")
	h := NewHandler(Deps{Pool: nil, JWTSigner: s})
	for _, path := range []string{
		"/api/v1/courses/cs101/analytics/insights",
		"/api/v1/courses/cs101/analytics/cross-section",
		"/api/v1/courses/cs101/analytics/insights/dismiss",
		"/api/v1/courses/cs101/analytics/insights/refresh",
	} {
		rr := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, path, nil)
		// No bearer — auth runs first, expect 401, not 404
		h.ServeHTTP(rr, r)
		if rr.Code == http.StatusNotFound {
			t.Fatalf("route not registered: %s", path)
		}
	}
}

func TestInsights_FeatureDisabled_NotFound(t *testing.T) {
	s := auth.NewJWTSigner("01234567890123456789012345678901")
	h := NewHandler(Deps{Pool: nil, JWTSigner: s})
	rr := httptest.NewRecorder()
	// Feature disabled by default (InstructorInsightsEnabled = false)
	r := httptest.NewRequest(http.MethodGet, "/api/v1/courses/cs101/analytics/insights", nil)
	r.Header.Set("Authorization", "Bearer x") // invalid token → 401 before feature check
	h.ServeHTTP(rr, r)
	if rr.Code == http.StatusNotFound {
		t.Fatalf("unexpected 404: auth should run first")
	}
}

func TestCrossSection_MethodNotAllowed(t *testing.T) {
	s := auth.NewJWTSigner("01234567890123456789012345678901")
	h := NewHandler(Deps{Pool: nil, JWTSigner: s})
	rr := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/api/v1/courses/cs101/analytics/cross-section", nil)
	r.Header.Set("Authorization", "Bearer x")
	h.ServeHTTP(rr, r)
	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("want 405, got %d", rr.Code)
	}
}
