package httpserver

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/lextures/lextures/server/internal/auth"
	"github.com/lextures/lextures/server/internal/config"
)

func bookstoreTestToken(t *testing.T, signer *auth.JWTSigner) string {
	t.Helper()
	tok, err := signer.Sign(context.Background(), "00000000-0000-0000-0000-000000000001", "u@test.invalid", "", "", nil)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	return tok
}

var bookstoreAuthRoutes = []struct {
	method string
	path   string
}{
	{http.MethodPost, "/api/v1/courses/test-course/structure/modules/00000000-0000-0000-0000-000000000002/textbook-resources"},
	{http.MethodGet, "/api/v1/courses/test-course/textbook-resources/00000000-0000-0000-0000-000000000002"},
	{http.MethodPatch, "/api/v1/courses/test-course/textbook-resources/00000000-0000-0000-0000-000000000002"},
	{http.MethodPost, "/api/v1/courses/test-course/textbook-resources/00000000-0000-0000-0000-000000000002/access"},
	{http.MethodGet, "/api/v1/courses/test-course/textbook-launch-events"},
	{http.MethodGet, "/api/v1/courses/test-course/inclusive-access"},
	{http.MethodPost, "/api/v1/courses/test-course/inclusive-access"},
}

var bookstoreAdminRoutes = []struct {
	method string
	path   string
}{
	{http.MethodGet, "/api/v1/admin/bookstore/config"},
	{http.MethodPost, "/api/v1/admin/bookstore/config"},
}

func TestBookstoreRoutes_NotFound404(t *testing.T) {
	signer := auth.NewJWTSigner("01234567890123456789012345678901")
	cfg := config.Config{FFBookstoreIntegration: true}
	h := NewHandler(Deps{Pool: nil, JWTSigner: signer, Config: cfg})
	tok := bookstoreTestToken(t, signer)

	allRoutes := append(append([]struct {
		method string
		path   string
	}{}, bookstoreAuthRoutes...), bookstoreAdminRoutes...)
	for _, c := range allRoutes {
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

func TestBookstoreRoutes_Unauthenticated401(t *testing.T) {
	signer := auth.NewJWTSigner("01234567890123456789012345678901")
	cfg := config.Config{FFBookstoreIntegration: true}
	h := NewHandler(Deps{Pool: nil, JWTSigner: signer, Config: cfg})

	for _, c := range bookstoreAuthRoutes {
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

func TestBookstoreRoutes_FeatureOff_Returns501(t *testing.T) {
	signer := auth.NewJWTSigner("01234567890123456789012345678901")
	cfg := config.Config{FFBookstoreIntegration: false}
	h := NewHandler(Deps{Pool: nil, JWTSigner: signer, Config: cfg})
	tok := bookstoreTestToken(t, signer)

	for _, c := range bookstoreAuthRoutes {
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

func TestValidBookstoreProvider(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		{"vitalsource", true},
		{"redshelf", true},
		{"VitalSource", false}, // case-sensitive — callers lowercase first
		{"chegg", false},
		{"", false},
	}
	for _, c := range cases {
		if got := validBookstoreProvider(c.in); got != c.want {
			t.Errorf("validBookstoreProvider(%q) = %v; want %v", c.in, got, c.want)
		}
	}
}
