package httpserver

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/lextures/lextures/server/internal/config"
)

func TestGetOERSearch_Unauthorized(t *testing.T) {
	h := NewHandler(Deps{Pool: nil, JWTSigner: nil})
	rr := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api/v1/oer/search?provider=oer_commons&q=test", nil)
	h.ServeHTTP(rr, r)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rr.Code)
	}
}

func TestGetOERSearch_FeatureDisabled(t *testing.T) {
	cfg := config.Load()
	cfg.OERLibraryEnabled = false
	h := NewHandler(Deps{Pool: nil, JWTSigner: nil, Config: cfg})
	rr := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api/v1/oer/search?provider=oer_commons&q=test", nil)
	h.ServeHTTP(rr, r)
	if rr.Code != http.StatusUnauthorized && rr.Code != http.StatusNotFound {
		t.Fatalf("expected 401 or 404 when feature off, got %d", rr.Code)
	}
}

func TestGetOERProviders_OptionsReturns204(t *testing.T) {
	h := NewHandler(Deps{Pool: nil, JWTSigner: nil})
	rr := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodOptions, "/api/v1/oer/providers", nil)
	h.ServeHTTP(rr, r)
	if rr.Code != http.StatusNoContent {
		t.Fatalf("OPTIONS oer/providers: expected 204, got %d", rr.Code)
	}
}

func TestPutAdminOERProvider_Unauthorized(t *testing.T) {
	h := NewHandler(Deps{Pool: nil, JWTSigner: nil})
	rr := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPut, "/api/v1/admin/oer-providers/merlot", nil)
	h.ServeHTTP(rr, r)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rr.Code)
	}
}
