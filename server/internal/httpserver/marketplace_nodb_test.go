package httpserver

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/lextures/lextures/server/internal/auth"
	"github.com/lextures/lextures/server/internal/config"
)

func TestMarketplace_FeatureOff(t *testing.T) {
	d := Deps{Config: config.Config{FFMarketplace: false}}
	h := NewHandler(d)

	paths := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/api/v1/marketplace/apps"},
		{http.MethodGet, "/api/v1/marketplace/apps/some-slug"},
		{http.MethodGet, "/oauth/authorize"},
		{http.MethodPost, "/oauth/token"},
		{http.MethodPost, "/oauth/revoke"},
		{http.MethodGet, "/api/v1/developer/apps"},
		{http.MethodPost, "/api/v1/developer/apps"},
		{http.MethodGet, "/api/v1/admin/marketplace/installed"},
		{http.MethodDelete, "/api/v1/admin/marketplace/installed/00000000-0000-4000-8000-000000000001"},
	}
	for _, tc := range paths {
		req := httptest.NewRequest(tc.method, tc.path, nil)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		if w.Code != http.StatusNotImplemented {
			t.Errorf("%s %s: want 501 got %d", tc.method, tc.path, w.Code)
		}
	}
}

func TestMarketplacePublic_NoAuth_FeatureOn(t *testing.T) {
	// Public marketplace listing should 501 when feature is off but succeed (return something)
	// when feature is on — even without a database (returns 500 because Pool is nil).
	d := Deps{Config: config.Config{FFMarketplace: true}}
	h := NewHandler(d)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/marketplace/apps", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	// No pool → internal error, but NOT 501 (feature is on).
	if w.Code == http.StatusNotImplemented {
		t.Fatalf("GET /api/v1/marketplace/apps returned 501 even with feature on")
	}
}

func TestOAuthAuthorize_Unauthenticated(t *testing.T) {
	signer := auth.NewJWTSigner("01234567890123456789012345678901")
	d := Deps{Config: config.Config{FFMarketplace: true}, JWTSigner: signer}
	h := NewHandler(d)

	req := httptest.NewRequest(http.MethodGet, "/oauth/authorize?client_id=test&redirect_uri=https://example.com/cb&code_challenge=abc&code_challenge_method=S256", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("GET /oauth/authorize without session: want 401 got %d", w.Code)
	}
}

func TestOAuthToken_MissingGrantType(t *testing.T) {
	d := Deps{Config: config.Config{FFMarketplace: true}}
	h := NewHandler(d)

	req := httptest.NewRequest(http.MethodPost, "/oauth/token", nil)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("POST /oauth/token without grant_type: want 400 got %d", w.Code)
	}
}

func TestDeveloperApps_Unauthenticated(t *testing.T) {
	signer := auth.NewJWTSigner("01234567890123456789012345678901")
	d := Deps{Config: config.Config{FFMarketplace: true}, JWTSigner: signer}
	h := NewHandler(d)

	for _, path := range []string{"/api/v1/developer/apps", "/api/v1/admin/marketplace/installed"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		if w.Code != http.StatusUnauthorized {
			t.Errorf("GET %s without session: want 401 got %d", path, w.Code)
		}
	}
}
