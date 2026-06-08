package httpserver

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/lextures/lextures/server/internal/auth"
	"github.com/lextures/lextures/server/internal/config"
)

func catalogTestToken(t *testing.T, signer *auth.JWTSigner) string {
	t.Helper()
	tok, err := signer.Sign(context.Background(), "00000000-0000-0000-0000-000000000001", "u@test.invalid", "", "", nil)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	return tok
}

func TestCatalogRoutes_NotFound404(t *testing.T) {
	signer := auth.NewJWTSigner("01234567890123456789012345678901")
	cfg := config.Config{FFCatalogIntegration: true}
	h := NewHandler(Deps{Pool: nil, JWTSigner: signer, Config: cfg})
	tok := catalogTestToken(t, signer)

	cases := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/api/v1/catalog/sections"},
		{http.MethodGet, "/api/v1/catalog/sections/00000000-0000-0000-0000-000000000002"},
		{http.MethodGet, "/api/v1/catalog/schedule"},
		{http.MethodPost, "/api/v1/admin/catalog/sync"},
		{http.MethodGet, "/api/v1/admin/catalog/sync-status"},
		{http.MethodGet, "/api/v1/courses/C-TEST01/catalog-info"},
	}

	for _, c := range cases {
		t.Run(c.method+" "+c.path, func(t *testing.T) {
			req := httptest.NewRequest(c.method, c.path, nil)
			req.Header.Set("Authorization", "Bearer "+tok)
			rr := httptest.NewRecorder()
			h.ServeHTTP(rr, req)
			if rr.Code == http.StatusNotFound {
				t.Fatalf("expected route to be registered, got 404 for %s %s: %s",
					c.method, c.path, rr.Body.String())
			}
		})
	}
}

func TestCatalogRoutes_Unauthenticated401(t *testing.T) {
	signer := auth.NewJWTSigner("01234567890123456789012345678901")
	cfg := config.Config{FFCatalogIntegration: true}
	h := NewHandler(Deps{Pool: nil, JWTSigner: signer, Config: cfg})

	cases := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/api/v1/catalog/sections"},
		{http.MethodGet, "/api/v1/catalog/schedule"},
		{http.MethodPost, "/api/v1/admin/catalog/sync"},
	}

	for _, c := range cases {
		t.Run(c.method+" "+c.path, func(t *testing.T) {
			req := httptest.NewRequest(c.method, c.path, nil)
			rr := httptest.NewRecorder()
			h.ServeHTTP(rr, req)
			if rr.Code != http.StatusUnauthorized {
				t.Fatalf("expected 401 without auth, got %d for %s %s",
					rr.Code, c.method, c.path)
			}
		})
	}
}

func TestCatalogRoutes_FeatureOff501(t *testing.T) {
	signer := auth.NewJWTSigner("01234567890123456789012345678901")
	cfg := config.Config{FFCatalogIntegration: false}
	h := NewHandler(Deps{Pool: nil, JWTSigner: signer, Config: cfg})
	tok := catalogTestToken(t, signer)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/catalog/sections", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusNotImplemented {
		t.Fatalf("expected 501 when feature off, got %d: %s", rr.Code, rr.Body.String())
	}
}
