package httpserver

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/lextures/lextures/server/internal/auth"
	"github.com/lextures/lextures/server/internal/config"
)

func heLibraryTestToken(t *testing.T, signer *auth.JWTSigner) string {
	t.Helper()
	tok, err := signer.Sign(context.Background(), "00000000-0000-0000-0000-000000000001", "u@test.invalid", "", "", nil)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	return tok
}

var heLibraryAuthRoutes = []struct {
	method string
	path   string
}{
	{http.MethodGet, "/api/v1/library/search?q=test"},
	{http.MethodPost, "/api/v1/courses/test-course/structure/modules/00000000-0000-0000-0000-000000000002/library-resources"},
	{http.MethodGet, "/api/v1/courses/test-course/library-resources/00000000-0000-0000-0000-000000000002"},
	{http.MethodPatch, "/api/v1/courses/test-course/library-resources/00000000-0000-0000-0000-000000000002"},
	{http.MethodPost, "/api/v1/courses/test-course/library-resources/00000000-0000-0000-0000-000000000002/access"},
	{http.MethodGet, "/api/v1/courses/test-course/library-link-events"},
}

// Admin routes use adminRbacUser which requires pool; just check route registration.
var heLibraryAdminRoutes = []struct {
	method string
	path   string
}{
	{http.MethodGet, "/api/v1/admin/library/config"},
	{http.MethodPost, "/api/v1/admin/library/config"},
}

func TestHELibraryRoutes_NotFound404(t *testing.T) {
	signer := auth.NewJWTSigner("01234567890123456789012345678901")
	cfg := config.Config{FFLibraryIntegration: true}
	h := NewHandler(Deps{Pool: nil, JWTSigner: signer, Config: cfg})
	tok := heLibraryTestToken(t, signer)

	allRoutes := append(heLibraryAuthRoutes, heLibraryAdminRoutes...)
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

func TestHELibraryRoutes_Unauthenticated401(t *testing.T) {
	signer := auth.NewJWTSigner("01234567890123456789012345678901")
	cfg := config.Config{FFLibraryIntegration: true}
	h := NewHandler(Deps{Pool: nil, JWTSigner: signer, Config: cfg})

	for _, c := range heLibraryAuthRoutes {
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

func TestHELibraryRoutes_FeatureOff_Returns501(t *testing.T) {
	signer := auth.NewJWTSigner("01234567890123456789012345678901")
	cfg := config.Config{FFLibraryIntegration: false}
	h := NewHandler(Deps{Pool: nil, JWTSigner: signer, Config: cfg})
	tok := heLibraryTestToken(t, signer)

	for _, c := range heLibraryAuthRoutes {
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

func TestEZProxyRewrite(t *testing.T) {
	cases := []struct {
		prefix   string
		patterns []string
		input    string
		want     string
	}{
		{
			prefix:   "https://ezproxy.university.edu",
			patterns: []string{"journals.sagepub.com"},
			input:    "https://journals.sagepub.com/article/123",
			want:     "https://ezproxy.university.edu/login?url=https://journals.sagepub.com/article/123",
		},
		{
			prefix:   "https://ezproxy.university.edu",
			patterns: []string{"*.springer.com"},
			input:    "https://link.springer.com/book/10.1007/978-3-030-01234-5",
			want:     "https://ezproxy.university.edu/login?url=https://link.springer.com/book/10.1007/978-3-030-01234-5",
		},
		{
			prefix:   "https://ezproxy.university.edu",
			patterns: []string{"journals.sagepub.com"},
			input:    "https://external-site.com/page",
			want:     "https://external-site.com/page",
		},
		{
			prefix:   "",
			patterns: []string{"journals.sagepub.com"},
			input:    "https://journals.sagepub.com/article/123",
			want:     "https://journals.sagepub.com/article/123",
		},
	}
	for _, c := range cases {
		got := rewriteEZProxy(c.prefix, c.patterns, c.input)
		if got != c.want {
			t.Errorf("rewriteEZProxy(%q, %v, %q) = %q; want %q",
				c.prefix, c.patterns, c.input, got, c.want)
		}
	}
}
