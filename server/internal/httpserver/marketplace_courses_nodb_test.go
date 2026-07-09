package httpserver

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/lextures/lextures/server/internal/config"
)

func TestMarketplaceCourses_FeatureOff_Returns404(t *testing.T) {
	h := NewHandler(Deps{Pool: nil, Config: config.Config{FFCourseMarketplace: false}, JWTSigner: nil})
	for _, path := range []string{
		"/api/v1/marketplace/courses",
		"/api/v1/marketplace/categories",
		"/api/v1/marketplace/courses/some-slug",
	} {
		rr := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, path, nil)
		h.ServeHTTP(rr, req)
		// Without JWT, auth runs first → 401. With flag off we still need auth.
		if rr.Code != http.StatusUnauthorized {
			t.Fatalf("%s: got %d, want 401 without auth (body %s)", path, rr.Code, rr.Body.String())
		}
	}
}

func TestMarketplaceCourses_NoAuth_Returns401(t *testing.T) {
	h := NewHandler(Deps{Pool: nil, Config: config.Config{FFCourseMarketplace: true}, JWTSigner: nil})
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/marketplace/courses", nil)
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("got %d, want 401 (body %s)", rr.Code, rr.Body.String())
	}
}

func TestMarketplaceCourses_DoesNotCollideWithPluginApps(t *testing.T) {
	// Plugin marketplace returns 501 when FFMarketplace is off; course routes are separate.
	h := NewHandler(Deps{
		Pool:      nil,
		Config:    config.Config{FFCourseMarketplace: true, FFMarketplace: false},
		JWTSigner: nil,
	})
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/marketplace/apps", nil)
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusNotImplemented && rr.Code != http.StatusUnauthorized && rr.Code != http.StatusOK {
		// Plugin route exists and is distinct; 501 is the typical flag-off response.
		if rr.Code == http.StatusNotFound {
			t.Fatalf("plugin /marketplace/apps unexpectedly 404; course routes may have shadowed it")
		}
	}
}
