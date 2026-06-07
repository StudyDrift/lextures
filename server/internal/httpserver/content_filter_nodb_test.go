package httpserver

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/lextures/lextures/server/internal/auth"
	"github.com/lextures/lextures/server/internal/config"
)

func contentFilterTestToken(t *testing.T, signer *auth.JWTSigner) string {
	t.Helper()
	tok, err := signer.Sign(context.Background(), "00000000-0000-0000-0000-000000000001", "u@test.invalid", "", "", nil)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	return tok
}

func TestContentFilterAllowlist_Public(t *testing.T) {
	h := NewHandler(Deps{Pool: nil})
	req := httptest.NewRequest(http.MethodGet, "/.well-known/content-filter-allowlist.json", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	if ct := rr.Header().Get("Content-Type"); ct != "application/json; charset=utf-8" {
		t.Fatalf("expected json content-type, got %q", ct)
	}
	if !strings.Contains(rr.Body.String(), "first_party_domains") {
		t.Fatalf("expected allowlist JSON body, got %s", rr.Body.String())
	}
}

func TestContentFilterRoutes_Registered(t *testing.T) {
	signer := auth.NewJWTSigner("01234567890123456789012345678901")
	cfg := config.Config{FFContentFilterIntegration: true}
	h := NewHandler(Deps{Pool: nil, JWTSigner: signer, Config: cfg})
	tok := contentFilterTestToken(t, signer)

	cases := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/api/v1/orgs/00000000-0000-0000-0000-000000000002/settings/content-filter"},
		{http.MethodPatch, "/api/v1/orgs/00000000-0000-0000-0000-000000000002/settings/content-filter"},
		{http.MethodPost, "/api/v1/content-filter/activity"},
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

func TestContentFilterRoutes_FeatureOff501(t *testing.T) {
	signer := auth.NewJWTSigner("01234567890123456789012345678901")
	cfg := config.Config{FFContentFilterIntegration: false}
	h := NewHandler(Deps{Pool: nil, JWTSigner: signer, Config: cfg})
	tok := contentFilterTestToken(t, signer)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/orgs/00000000-0000-0000-0000-000000000002/settings/content-filter", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusNotImplemented {
		t.Fatalf("expected 501 when feature off, got %d: %s", rr.Code, rr.Body.String())
	}
}
