package httpserver

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/lextures/lextures/server/internal/auth"
	"github.com/lextures/lextures/server/internal/config"
)

func sbgTestToken(t *testing.T, signer *auth.JWTSigner) string {
	t.Helper()
	tok, err := signer.Sign(context.Background(), "00000000-0000-0000-0000-000000000001", "u@test.invalid", "", "", nil)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	return tok
}

func TestSBGRoutes_NotFound404(t *testing.T) {
	signer := auth.NewJWTSigner("01234567890123456789012345678901")
	h := NewHandler(Deps{Pool: nil, JWTSigner: signer, Config: config.Config{}})
	tok := sbgTestToken(t, signer)

	cases := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/api/v1/admin/orgs/00000000-0000-0000-0000-000000000001/sbg/standard-domains"},
		{http.MethodPost, "/api/v1/admin/orgs/00000000-0000-0000-0000-000000000001/sbg/standard-domains"},
		{http.MethodGet, "/api/v1/admin/orgs/00000000-0000-0000-0000-000000000001/sbg/mastery-scale"},
		{http.MethodPut, "/api/v1/admin/orgs/00000000-0000-0000-0000-000000000001/sbg/mastery-scale"},
		{http.MethodPost, "/api/v1/admin/orgs/00000000-0000-0000-0000-000000000001/sbg/standards/import"},
		{http.MethodGet, "/api/v1/courses/COURSE01/sbg/standards"},
		{http.MethodPost, "/api/v1/sbg/mastery-scores"},
		{http.MethodGet, "/api/v1/courses/COURSE01/sbg/heatmap/Q1-2026"},
		{http.MethodGet, "/api/v1/students/00000000-0000-0000-0000-000000000001/sbg/Q1-2026"},
	}

	for _, c := range cases {
		t.Run(c.method+" "+c.path, func(t *testing.T) {
			req := httptest.NewRequest(c.method, c.path, nil)
			req.Header.Set("Authorization", "Bearer "+tok)
			rr := httptest.NewRecorder()
			h.ServeHTTP(rr, req)
			if rr.Code == http.StatusNotFound {
				t.Fatalf("route not registered — got 404 for %s %s: %s",
					c.method, c.path, rr.Body.String())
			}
		})
	}
}

func TestSBGRoutes_Unauthenticated401(t *testing.T) {
	signer := auth.NewJWTSigner("01234567890123456789012345678901")
	h := NewHandler(Deps{Pool: nil, JWTSigner: signer, Config: config.Config{}})

	cases := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/api/v1/admin/orgs/00000000-0000-0000-0000-000000000001/sbg/standard-domains"},
		{http.MethodPost, "/api/v1/admin/orgs/00000000-0000-0000-0000-000000000001/sbg/standard-domains"},
		{http.MethodGet, "/api/v1/admin/orgs/00000000-0000-0000-0000-000000000001/sbg/mastery-scale"},
		{http.MethodPut, "/api/v1/admin/orgs/00000000-0000-0000-0000-000000000001/sbg/mastery-scale"},
		{http.MethodPost, "/api/v1/admin/orgs/00000000-0000-0000-0000-000000000001/sbg/standards/import"},
		{http.MethodGet, "/api/v1/courses/COURSE01/sbg/standards"},
		{http.MethodPost, "/api/v1/sbg/mastery-scores"},
		{http.MethodGet, "/api/v1/courses/COURSE01/sbg/heatmap/Q1-2026"},
		{http.MethodGet, "/api/v1/students/00000000-0000-0000-0000-000000000001/sbg/Q1-2026"},
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
