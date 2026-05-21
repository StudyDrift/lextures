package httpserver

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// These tests verify caption route registration and auth guards without a database.

func TestHandleListCaptions_Unauthenticated(t *testing.T) {
	h := NewHandler(Deps{})
	r := httptest.NewRequest(http.MethodGet, "/api/v1/files/00000000-0000-0000-0000-000000000001/captions", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, r)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 got %d", rr.Code)
	}
}

func TestHandleListCaptions_InvalidUUID(t *testing.T) {
	h := NewHandler(Deps{})
	r := httptest.NewRequest(http.MethodGet, "/api/v1/files/not-a-uuid/captions", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, r)
	// Route won't match (chi pattern requires valid segment); 404 is expected
	if rr.Code == http.StatusOK {
		t.Fatalf("expected non-200 for invalid UUID path, got 200")
	}
}

func TestHandleGetCaptionVTT_Unauthenticated(t *testing.T) {
	h := NewHandler(Deps{})
	r := httptest.NewRequest(http.MethodGet,
		"/api/v1/files/00000000-0000-0000-0000-000000000001/captions/00000000-0000-0000-0000-000000000002/vtt", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, r)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 got %d", rr.Code)
	}
}

func TestHandleUpdateCaption_Unauthenticated(t *testing.T) {
	h := NewHandler(Deps{})
	r := httptest.NewRequest(http.MethodPut,
		"/api/v1/files/00000000-0000-0000-0000-000000000001/captions/00000000-0000-0000-0000-000000000002", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, r)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 got %d", rr.Code)
	}
}

func TestHandleRetriggerCaption_Unauthenticated(t *testing.T) {
	h := NewHandler(Deps{})
	r := httptest.NewRequest(http.MethodPost,
		"/api/v1/files/00000000-0000-0000-0000-000000000001/captions/retrigger", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, r)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 got %d", rr.Code)
	}
}

func TestHandleCaptionCoverageReport_Unauthenticated(t *testing.T) {
	h := NewHandler(Deps{})
	r := httptest.NewRequest(http.MethodGet, "/api/v1/reports/caption-coverage", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, r)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 got %d", rr.Code)
	}
}

func TestHandleCaptionCoverageReport_NonAdmin(t *testing.T) {
	// Without an admin JWT, the endpoint must reject the request.
	h := NewHandler(Deps{})
	r := httptest.NewRequest(http.MethodGet, "/api/v1/reports/caption-coverage", nil)
	// No auth header → still 401
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, r)
	if rr.Code == http.StatusOK {
		t.Fatalf("expected non-200 for unauthenticated request, got 200")
	}
}

// Ensure the caption routes are registered (route not 405 or unknown).
func TestCaptionRoutesRegistered(t *testing.T) {
	h := NewHandler(Deps{})

	routes := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/api/v1/files/00000000-0000-0000-0000-000000000001/captions"},
		{http.MethodGet, "/api/v1/files/00000000-0000-0000-0000-000000000001/captions/00000000-0000-0000-0000-000000000002/vtt"},
		{http.MethodPut, "/api/v1/files/00000000-0000-0000-0000-000000000001/captions/00000000-0000-0000-0000-000000000002"},
		{http.MethodPost, "/api/v1/files/00000000-0000-0000-0000-000000000001/captions/retrigger"},
		{http.MethodGet, "/api/v1/reports/caption-coverage"},
	}

	for _, rt := range routes {
		req := httptest.NewRequest(rt.method, rt.path, nil)
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, req)
		// Routes must respond with something other than 404 (not found means route not registered).
		// 401 is expected since no JWT is supplied.
		if rr.Code == http.StatusNotFound {
			t.Errorf("%s %s: got 404, route not registered", rt.method, rt.path)
		}
	}
}
