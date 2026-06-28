package httpserver

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRequestIsAuthenticated(t *testing.T) {
	t.Run("public path without auth", func(t *testing.T) {
		r, _ := http.NewRequest(http.MethodGet, "/api/v1/public/catalog/courses", nil)
		if requestIsAuthenticated(r) {
			t.Fatal("expected unauthenticated public path")
		}
	})
	t.Run("bearer token", func(t *testing.T) {
		r, _ := http.NewRequest(http.MethodGet, "/api/v1/me", nil)
		r.Header.Set("Authorization", "Bearer tok")
		if !requestIsAuthenticated(r) {
			t.Fatal("expected authenticated with bearer")
		}
	})
}

func TestAuthenticatedNoStoreMiddleware(t *testing.T) {
	h := authenticatedNoStoreMiddleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"ok": "1"})
	}))
	r, _ := http.NewRequest(http.MethodGet, "/api/v1/me", nil)
	r.Header.Set("Authorization", "Bearer tok")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, r)
	if got := rec.Header().Get("Cache-Control"); got != "no-store" {
		t.Fatalf("Cache-Control=%q want no-store", got)
	}
}
