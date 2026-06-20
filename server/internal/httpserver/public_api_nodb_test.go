package httpserver

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/lextures/lextures/server/internal/config"
)

func TestPublicAPI_FeatureOff_Returns503(t *testing.T) {
	t.Parallel()
	h := NewHandler(Deps{Config: config.Config{FFPublicAPI: false}})
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/assignments", nil)
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("got %d, want 503", rr.Code)
	}
}

func TestPublicAPI_UnauthenticatedCourses_ProblemJSON(t *testing.T) {
	t.Parallel()
	h := NewHandler(Deps{Config: config.Config{FFPublicAPI: true}})
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/courses", nil)
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("got %d, want 401", rr.Code)
	}
	if ct := rr.Header().Get("Content-Type"); ct != "application/problem+json; charset=utf-8" {
		t.Fatalf("content-type: %q", ct)
	}
}

func TestPublicAPI_OpenAPISpec(t *testing.T) {
	t.Parallel()
	h := NewHandler(Deps{})
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/openapi.json", nil)
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("got %d", rr.Code)
	}
}

func TestPublicAPI_DocsDisabled(t *testing.T) {
	t.Parallel()
	h := NewHandler(Deps{Config: config.Config{EnableAPIDocs: false, FFAPIDocs: false}})
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/docs", nil)
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("got %d, want 404", rr.Code)
	}
}

func TestPublicAPI_DocsEnabledViaEnv(t *testing.T) {
	t.Parallel()
	h := NewHandler(Deps{Config: config.Config{EnableAPIDocs: true}})
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/docs", nil)
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("got %d, want 200", rr.Code)
	}
}
