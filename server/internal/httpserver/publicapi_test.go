package httpserver

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/lextures/lextures/server/internal/config"
)

func TestPublicAPI_FeatureOff_Returns503(t *testing.T) {
	h := NewHandler(Deps{Pool: nil, Config: config.Config{FFPublicAPI: false}})
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/courses", nil)
	req.Header.Set("Authorization", "Bearer ltk_testtoken")
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("got %d body %s", rr.Code, rr.Body.String())
	}
}

func TestPublicAPI_Unauthenticated_Returns401Problem(t *testing.T) {
	h := NewHandler(Deps{Pool: nil, Config: config.Config{FFPublicAPI: true}})
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/courses", nil)
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("got %d body %s", rr.Code, rr.Body.String())
	}
	if ct := rr.Header().Get("Content-Type"); ct != "application/problem+json; charset=utf-8" {
		t.Fatalf("content-type %q", ct)
	}
	var body map[string]any
	if err := json.NewDecoder(rr.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body["title"] != "Unauthorized" {
		t.Fatalf("body %v", body)
	}
}

func TestPublicAPI_OpenAPISpec_Public(t *testing.T) {
	h := NewHandler(Deps{Pool: nil, Config: config.Config{}})
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/openapi.json", nil)
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("got %d", rr.Code)
	}
	var doc map[string]any
	if err := json.NewDecoder(rr.Body).Decode(&doc); err != nil {
		t.Fatal(err)
	}
	if doc["openapi"] != "3.1.0" {
		t.Fatalf("version %v", doc["openapi"])
	}
}

func TestPublicAPI_Docs_FlagGated(t *testing.T) {
	h := NewHandler(Deps{Pool: nil, Config: config.Config{EnableAPIDocs: false}})
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/docs", nil)
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("got %d", rr.Code)
	}
}

func TestPublicAPI_Docs_Enabled(t *testing.T) {
	h := NewHandler(Deps{Pool: nil, Config: config.Config{EnableAPIDocs: true}})
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/docs", nil)
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("got %d", rr.Code)
	}
	if ct := rr.Header().Get("Content-Type"); ct != "text/html; charset=utf-8" {
		t.Fatalf("content-type %q", ct)
	}
}
