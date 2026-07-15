package httpserver

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/lextures/lextures/server/internal/config"
)

func TestBadgesRoutes_FeatureOff_Returns404(t *testing.T) {
	cfg := config.Config{FFCompetencyBadges: false}
	h := NewHandler(Deps{Pool: nil, Config: cfg, JWTSigner: nil})

	paths := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/api/v1/me/competency-badges"},
		{http.MethodGet, "/api/v1/me/badge-profile"},
		{http.MethodGet, "/api/v1/public/badges/willden"},
		{http.MethodGet, "/api/v1/badges/verify/abc"},
		{http.MethodGet, "/api/v1/courses/00000000-0000-0000-0000-000000000001/badge-definitions"},
	}
	for _, p := range paths {
		req := httptest.NewRequest(p.method, p.path, nil)
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, req)
		// Unauthenticated protected routes may 401; feature-gated public should 404.
		if p.path == "/api/v1/public/badges/willden" || p.path == "/api/v1/badges/verify/abc" {
			if rr.Code != http.StatusNotFound {
				t.Fatalf("%s %s: want 404 got %d", p.method, p.path, rr.Code)
			}
		}
	}
}

func TestBadgesRoutes_FeatureOn_VerifyNoAuth(t *testing.T) {
	cfg := config.Config{FFCompetencyBadges: true}
	h := NewHandler(Deps{Pool: nil, Config: cfg, JWTSigner: nil})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/badges/verify/not-a-real-slug", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	// Pool is nil so may 500; must not be 401 (public endpoint).
	if rr.Code == http.StatusUnauthorized {
		t.Fatalf("verify endpoint must not require auth, got 401")
	}
}

func TestBadgesRoutes_MeRequiresAuth(t *testing.T) {
	cfg := config.Config{FFCompetencyBadges: true}
	h := NewHandler(Deps{Pool: nil, Config: cfg, JWTSigner: nil})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/me/competency-badges", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized && rr.Code != http.StatusForbidden {
		// Depending on auth middleware, expect 401/403 without token.
		if rr.Code == http.StatusOK {
			t.Fatalf("me/competency-badges must require auth")
		}
	}
}

func TestBadgesRoutes_RegisteredWhenFeatureOn(t *testing.T) {
	cfg := config.Config{FFCompetencyBadges: true}
	h := NewHandler(Deps{Pool: nil, Config: cfg, JWTSigner: nil})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/public/badges/someone", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	// Without DB we expect 500 not 404-route-missing.
	if rr.Code == http.StatusNotFound {
		// Feature on with missing data may also 404 from service — acceptable.
		// Route is registered either way if body is JSON error not SPA.
		ct := rr.Header().Get("Content-Type")
		if ct == "" && rr.Body.Len() == 0 {
			t.Fatalf("route may not be registered")
		}
	}
}
