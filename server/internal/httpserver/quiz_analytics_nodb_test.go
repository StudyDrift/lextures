package httpserver

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandleGetQuizAnalytics_Unauthorized(t *testing.T) {
	h := NewHandler(Deps{})
	r := httptest.NewRequest(http.MethodGet, baseItemPath+"/analytics", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, r)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rr.Code)
	}
}

func TestHandleGetQuizAnalytics_Options(t *testing.T) {
	h := NewHandler(Deps{})
	r := httptest.NewRequest(http.MethodOptions, baseItemPath+"/analytics", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, r)
	if rr.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", rr.Code)
	}
}

func TestHandleGetQuizAnalytics_MethodNotAllowed(t *testing.T) {
	h := NewHandler(Deps{})
	r := httptest.NewRequest(http.MethodPost, baseItemPath+"/analytics", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, r)
	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", rr.Code)
	}
}
