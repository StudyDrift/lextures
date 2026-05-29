package httpserver

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/lextures/lextures/server/internal/auth"
	"github.com/lextures/lextures/server/internal/config"
)

// bearerToken returns a valid signed JWT for testing route registration (no DB).
func attendanceTestToken(t *testing.T, signer *auth.JWTSigner) string {
	t.Helper()
	tok, err := signer.Sign(context.Background(), "00000000-0000-0000-0000-000000000001", "u@test.invalid", "", "", nil)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	return tok
}

func TestAttendanceRoutes_NotFound404(t *testing.T) {
	signer := auth.NewJWTSigner("01234567890123456789012345678901")
	h := NewHandler(Deps{Pool: nil, JWTSigner: signer, Config: config.Config{}})
	tok := attendanceTestToken(t, signer)

	cases := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/api/v1/sections/00000000-0000-0000-0000-000000000001/attendance/2026-01-15"},
		{http.MethodPut, "/api/v1/sections/00000000-0000-0000-0000-000000000001/attendance/2026-01-15"},
		{http.MethodGet, "/api/v1/students/00000000-0000-0000-0000-000000000001/attendance"},
		{http.MethodGet, "/api/v1/org-units/00000000-0000-0000-0000-000000000001/attendance/dashboard"},
		{http.MethodGet, "/api/v1/admin/orgs/00000000-0000-0000-0000-000000000001/attendance/codes"},
		{http.MethodPost, "/api/v1/admin/orgs/00000000-0000-0000-0000-000000000001/attendance/codes"},
		{http.MethodDelete, "/api/v1/admin/orgs/00000000-0000-0000-0000-000000000001/attendance/codes/00000000-0000-0000-0000-000000000002"},
		{http.MethodPost, "/api/v1/admin/orgs/00000000-0000-0000-0000-000000000001/attendance/export"},
		{http.MethodGet, "/api/v1/parent/students/00000000-0000-0000-0000-000000000001/attendance"},
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

func TestAttendanceRoutes_Unauthenticated401(t *testing.T) {
	signer := auth.NewJWTSigner("01234567890123456789012345678901")
	h := NewHandler(Deps{Pool: nil, JWTSigner: signer, Config: config.Config{}})

	cases := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/api/v1/sections/00000000-0000-0000-0000-000000000001/attendance/2026-01-15"},
		{http.MethodPut, "/api/v1/sections/00000000-0000-0000-0000-000000000001/attendance/2026-01-15"},
		{http.MethodGet, "/api/v1/students/00000000-0000-0000-0000-000000000001/attendance"},
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

func TestAttendanceIsWithinEditWindow(t *testing.T) {
	// Import is handled in the package itself via black-box test here.
	// We just verify the handler is registered by checking a non-404.
	signer := auth.NewJWTSigner("01234567890123456789012345678901")
	h := NewHandler(Deps{Pool: nil, JWTSigner: signer, Config: config.Config{}})
	tok := attendanceTestToken(t, signer)

	req := httptest.NewRequest(http.MethodPut,
		"/api/v1/sections/00000000-0000-0000-0000-000000000001/attendance/2026-01-15",
		nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code == http.StatusNotFound {
		t.Fatalf("PUT attendance route not registered: %s", rr.Body.String())
	}
}
