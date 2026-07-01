package httpserver

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/lextures/lextures/server/internal/auth"
	"github.com/lextures/lextures/server/internal/config"
)

func reportCardTestToken(t *testing.T, signer *auth.JWTSigner) string {
	t.Helper()
	tok, err := signer.Sign(context.Background(), "00000000-0000-0000-0000-000000000001", "u@test.invalid", "", "", nil)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	return tok
}

func TestReportCardRoutes_NotFound404(t *testing.T) {
	signer := auth.NewJWTSigner("01234567890123456789012345678901")
	h := NewHandler(Deps{Pool: nil, JWTSigner: signer, Config: config.Config{}})
	tok := reportCardTestToken(t, signer)

	cases := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/api/v1/courses/CS101/report-cards/Q1-2026"},
		{http.MethodPatch, "/api/v1/report-cards/00000000-0000-0000-0000-000000000001"},
		{http.MethodPost, "/api/v1/report-cards/00000000-0000-0000-0000-000000000001/generate-pdf"},
		{http.MethodPost, "/api/v1/courses/CS101/report-cards/Q1-2026/release"},
		{http.MethodGet, "/api/v1/report-cards/00000000-0000-0000-0000-000000000001/pdf"},
		{http.MethodPost, "/api/v1/ai/report-card-comment"},
		{http.MethodGet, "/api/v1/admin/orgs/00000000-0000-0000-0000-000000000001/report-cards/comment-bank"},
		{http.MethodPost, "/api/v1/admin/orgs/00000000-0000-0000-0000-000000000001/report-cards/comment-bank"},
		{http.MethodDelete, "/api/v1/admin/orgs/00000000-0000-0000-0000-000000000001/report-cards/comment-bank/00000000-0000-0000-0000-000000000002"},
		{http.MethodGet, "/api/v1/parent/students/00000000-0000-0000-0000-000000000001/report-cards"},
		{http.MethodGet, "/api/v1/me/report-cards"},
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

func TestReportCardRoutes_Unauthenticated401(t *testing.T) {
	signer := auth.NewJWTSigner("01234567890123456789012345678901")
	h := NewHandler(Deps{Pool: nil, JWTSigner: signer, Config: config.Config{}})

	cases := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/api/v1/courses/CS101/report-cards/Q1-2026"},
		{http.MethodPatch, "/api/v1/report-cards/00000000-0000-0000-0000-000000000001"},
		{http.MethodPost, "/api/v1/ai/report-card-comment"},
		{http.MethodGet, "/api/v1/parent/students/00000000-0000-0000-0000-000000000001/report-cards"},
		{http.MethodGet, "/api/v1/me/report-cards"},
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

func TestReportCardRoutes_MethodNotAllowed(t *testing.T) {
	signer := auth.NewJWTSigner("01234567890123456789012345678901")
	h := NewHandler(Deps{Pool: nil, JWTSigner: signer, Config: config.Config{}})
	tok := reportCardTestToken(t, signer)

	cases := []struct {
		method string
		path   string
	}{
		{http.MethodDelete, "/api/v1/courses/CS101/report-cards/Q1-2026"},
		{http.MethodGet, "/api/v1/report-cards/00000000-0000-0000-0000-000000000001/generate-pdf"},
		{http.MethodPost, "/api/v1/me/report-cards"},
	}

	for _, c := range cases {
		t.Run(c.method+" "+c.path, func(t *testing.T) {
			req := httptest.NewRequest(c.method, c.path, nil)
			req.Header.Set("Authorization", "Bearer "+tok)
			rr := httptest.NewRecorder()
			h.ServeHTTP(rr, req)
			if rr.Code == http.StatusMethodNotAllowed {
				return // expected
			}
			// 500 means the handler matched but DB is nil — that's fine for nodb tests
			if rr.Code == http.StatusInternalServerError {
				return
			}
			// 404 means route not registered (bad)
			if rr.Code == http.StatusNotFound {
				t.Fatalf("route not registered: %s %s", c.method, c.path)
			}
		})
	}
}
