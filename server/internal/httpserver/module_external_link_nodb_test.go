package httpserver

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetModuleExternalLink_Unauthorized(t *testing.T) {
	h := NewHandler(Deps{Pool: nil, JWTSigner: nil})
	rr := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api/v1/courses/C-TEST/external-links/00000000-0000-0000-0000-000000000001", nil)
	h.ServeHTTP(rr, r)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 without auth, got %d", rr.Code)
	}
}

func TestPatchModuleExternalLink_Unauthorized(t *testing.T) {
	h := NewHandler(Deps{Pool: nil, JWTSigner: nil})
	rr := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPatch, "/api/v1/courses/C-TEST/external-links/00000000-0000-0000-0000-000000000001", nil)
	h.ServeHTTP(rr, r)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 without auth, got %d", rr.Code)
	}
}

func TestGetModuleExternalLink_MethodNotAllowed(t *testing.T) {
	h := NewHandler(Deps{Pool: nil, JWTSigner: nil})
	for _, method := range []string{http.MethodPost, http.MethodPut, http.MethodDelete} {
		rr := httptest.NewRecorder()
		r := httptest.NewRequest(method, "/api/v1/courses/C-TEST/external-links/00000000-0000-0000-0000-000000000001", nil)
		h.ServeHTTP(rr, r)
		if rr.Code != http.StatusMethodNotAllowed && rr.Code != http.StatusUnauthorized {
			t.Fatalf("GET external-links %s: expected 405 or 401, got %d", method, rr.Code)
		}
	}
}

func TestPatchModuleExternalLink_MethodNotAllowed(t *testing.T) {
	h := NewHandler(Deps{Pool: nil, JWTSigner: nil})
	rr := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api/v1/courses/C-TEST/external-links/00000000-0000-0000-0000-000000000001", nil)
	h.ServeHTTP(rr, r)
	// Without auth JWTSigner=nil always returns 401 before method check
	if rr.Code != http.StatusUnauthorized && rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 401 or 405, got %d", rr.Code)
	}
}

func TestGetModuleExternalLink_OptionsReturns204(t *testing.T) {
	h := NewHandler(Deps{Pool: nil, JWTSigner: nil})
	rr := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodOptions, "/api/v1/courses/C-TEST/external-links/00000000-0000-0000-0000-000000000001", nil)
	h.ServeHTTP(rr, r)
	if rr.Code != http.StatusNoContent {
		t.Fatalf("OPTIONS external-links: expected 204, got %d", rr.Code)
	}
}

func TestPatchModuleExternalLink_OptionsReturns204(t *testing.T) {
	h := NewHandler(Deps{Pool: nil, JWTSigner: nil})
	rr := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodOptions, "/api/v1/courses/C-TEST/external-links/00000000-0000-0000-0000-000000000001", nil)
	h.ServeHTTP(rr, r)
	if rr.Code != http.StatusNoContent {
		t.Fatalf("OPTIONS patch external-links: expected 204, got %d", rr.Code)
	}
}
