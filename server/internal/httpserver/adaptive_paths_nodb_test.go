package httpserver

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/lextures/lextures/server/internal/auth"
	"github.com/lextures/lextures/server/internal/config"
)

func TestAdaptivePathRoutes_RegisteredRequireAuth(t *testing.T) {
	signer := auth.NewJWTSigner("01234567890123456789012345678901")
	d := Deps{Pool: nil, JWTSigner: signer, Config: config.Config{}}
	h := NewHandler(d)

	paths := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/api/v1/courses/demo/concepts-for-path"},
		{http.MethodGet, "/api/v1/courses/demo/structure/items/00000000-0000-0000-0000-000000000001/path-rules"},
		{http.MethodPost, "/api/v1/courses/demo/structure/items/00000000-0000-0000-0000-000000000001/path-rules"},
		{http.MethodDelete, "/api/v1/courses/demo/structure/items/00000000-0000-0000-0000-000000000001/path-rules/00000000-0000-0000-0000-000000000002"},
	}
	for _, p := range paths {
		req := httptest.NewRequest(p.method, p.path, nil)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		// Unauthenticated: must not 404 (route missing). Expect 401.
		if w.Code == http.StatusNotFound {
			t.Fatalf("%s %s: route not registered (404)", p.method, p.path)
		}
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("%s %s: want 401 got %d body=%s", p.method, p.path, w.Code, w.Body.String())
		}
	}
}
