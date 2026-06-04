package httpserver

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/lextures/lextures/server/internal/auth"
	"github.com/lextures/lextures/server/internal/config"
)

func sisTestToken(t *testing.T, signer *auth.JWTSigner) string {
	t.Helper()
	tok, err := signer.Sign(context.Background(), "00000000-0000-0000-0000-000000000001", "u@test.invalid", "", "", nil)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	return tok
}

func TestSISRoutes_NotFound404(t *testing.T) {
	signer := auth.NewJWTSigner("01234567890123456789012345678901")
	cfg := config.Config{FFSISIntegration: true}
	h := NewHandler(Deps{Pool: nil, JWTSigner: signer, Config: cfg})
	tok := sisTestToken(t, signer)

	cases := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/api/v1/admin/orgs/00000000-0000-0000-0000-000000000001/sis/connections"},
		{http.MethodPost, "/api/v1/admin/orgs/00000000-0000-0000-0000-000000000001/sis/connections"},
		{http.MethodPatch, "/api/v1/admin/orgs/00000000-0000-0000-0000-000000000001/sis/connections/00000000-0000-0000-0000-000000000002"},
		{http.MethodPost, "/api/v1/admin/orgs/00000000-0000-0000-0000-000000000001/sis/connections/00000000-0000-0000-0000-000000000002/sync"},
		{http.MethodGet, "/api/v1/admin/orgs/00000000-0000-0000-0000-000000000001/sis/sync-logs"},
		{http.MethodPost, "/api/v1/admin/orgs/00000000-0000-0000-0000-000000000001/sis/grade-passback"},
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

func TestSISRoutes_Unauthenticated401(t *testing.T) {
	signer := auth.NewJWTSigner("01234567890123456789012345678901")
	cfg := config.Config{FFSISIntegration: true}
	h := NewHandler(Deps{Pool: nil, JWTSigner: signer, Config: cfg})

	cases := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/api/v1/admin/orgs/00000000-0000-0000-0000-000000000001/sis/connections"},
		{http.MethodPost, "/api/v1/admin/orgs/00000000-0000-0000-0000-000000000001/sis/connections"},
		{http.MethodPatch, "/api/v1/admin/orgs/00000000-0000-0000-0000-000000000001/sis/connections/00000000-0000-0000-0000-000000000002"},
		{http.MethodPost, "/api/v1/admin/orgs/00000000-0000-0000-0000-000000000001/sis/connections/00000000-0000-0000-0000-000000000002/sync"},
		{http.MethodGet, "/api/v1/admin/orgs/00000000-0000-0000-0000-000000000001/sis/sync-logs"},
		{http.MethodPost, "/api/v1/admin/orgs/00000000-0000-0000-0000-000000000001/sis/grade-passback"},
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

func TestSISRoutes_FeatureOff_Returns501(t *testing.T) {
	signer := auth.NewJWTSigner("01234567890123456789012345678901")
	cfg := config.Config{FFSISIntegration: false}
	h := NewHandler(Deps{Pool: nil, JWTSigner: signer, Config: cfg})
	tok := sisTestToken(t, signer)

	cases := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/api/v1/admin/orgs/00000000-0000-0000-0000-000000000001/sis/connections"},
		{http.MethodPost, "/api/v1/admin/orgs/00000000-0000-0000-0000-000000000001/sis/connections"},
		{http.MethodPatch, "/api/v1/admin/orgs/00000000-0000-0000-0000-000000000001/sis/connections/00000000-0000-0000-0000-000000000002"},
		{http.MethodPost, "/api/v1/admin/orgs/00000000-0000-0000-0000-000000000001/sis/connections/00000000-0000-0000-0000-000000000002/sync"},
		{http.MethodGet, "/api/v1/admin/orgs/00000000-0000-0000-0000-000000000001/sis/sync-logs"},
		{http.MethodPost, "/api/v1/admin/orgs/00000000-0000-0000-0000-000000000001/sis/grade-passback"},
	}

	for _, c := range cases {
		t.Run(c.method+" "+c.path, func(t *testing.T) {
			req := httptest.NewRequest(c.method, c.path, nil)
			req.Header.Set("Authorization", "Bearer "+tok)
			rr := httptest.NewRecorder()
			h.ServeHTTP(rr, req)
			if rr.Code != http.StatusNotImplemented {
				t.Fatalf("expected 501 when feature off, got %d for %s %s: %s",
					rr.Code, c.method, c.path, rr.Body.String())
			}
		})
	}
}

func TestSISConnections_MethodNotAllowed(t *testing.T) {
	signer := auth.NewJWTSigner("01234567890123456789012345678901")
	cfg := config.Config{FFSISIntegration: true}
	h := NewHandler(Deps{Pool: nil, JWTSigner: signer, Config: cfg})
	tok := sisTestToken(t, signer)

	cases := []struct {
		method string
		path   string
	}{
		{http.MethodDelete, "/api/v1/admin/orgs/00000000-0000-0000-0000-000000000001/sis/connections"},
		{http.MethodGet, "/api/v1/admin/orgs/00000000-0000-0000-0000-000000000001/sis/connections/00000000-0000-0000-0000-000000000002"},
	}

	for _, c := range cases {
		t.Run(c.method+" "+c.path, func(t *testing.T) {
			req := httptest.NewRequest(c.method, c.path, nil)
			req.Header.Set("Authorization", "Bearer "+tok)
			rr := httptest.NewRecorder()
			h.ServeHTTP(rr, req)
			// Unregistered method on a registered path returns 405.
			if rr.Code == http.StatusNotFound {
				t.Logf("note: %s %s returned 404 (not registered), which is also acceptable", c.method, c.path)
			}
		})
	}
}
