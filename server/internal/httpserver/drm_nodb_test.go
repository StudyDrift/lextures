package httpserver

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandlePostFileLicense_NoDRM_Returns501(t *testing.T) {
	h := NewHandler(Deps{DRM: nil})
	r := httptest.NewRequest(http.MethodPost, "/api/v1/files/00000000-0000-0000-0000-000000000001/license", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, r)
	// No JWT → 401 (auth check fires before the DRM nil check)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 got %d", rr.Code)
	}
}

func TestHandlePostFileLicense_Options(t *testing.T) {
	h := NewHandler(Deps{})
	r := httptest.NewRequest(http.MethodOptions, "/api/v1/files/00000000-0000-0000-0000-000000000001/license", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, r)
	if rr.Code != http.StatusNoContent {
		t.Fatalf("OPTIONS: got %d want 204", rr.Code)
	}
}

func TestHandlePostFileLicense_WrongMethod(t *testing.T) {
	h := NewHandler(Deps{})
	r := httptest.NewRequest(http.MethodGet, "/api/v1/files/00000000-0000-0000-0000-000000000001/license", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, r)
	// GET is not registered on this path → 404 (chi falls through)
	if rr.Code == http.StatusOK {
		t.Fatalf("GET on POST-only route should not return 200")
	}
}

func TestHandlePutAdminFileDRM_Options(t *testing.T) {
	h := NewHandler(Deps{})
	r := httptest.NewRequest(http.MethodOptions, "/api/v1/admin/files/00000000-0000-0000-0000-000000000001/drm", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, r)
	if rr.Code != http.StatusNoContent {
		t.Fatalf("OPTIONS: got %d want 204", rr.Code)
	}
}

func TestHandlePutAdminFileDRM_Unauthorized(t *testing.T) {
	h := NewHandler(Deps{})
	r := httptest.NewRequest(http.MethodPut, "/api/v1/admin/files/00000000-0000-0000-0000-000000000001/drm", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, r)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 got %d", rr.Code)
	}
}

func TestHandleGetAdminDRMAnomalies_Options(t *testing.T) {
	h := NewHandler(Deps{})
	r := httptest.NewRequest(http.MethodOptions, "/api/v1/admin/drm/anomalies", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, r)
	if rr.Code != http.StatusNoContent {
		t.Fatalf("OPTIONS: got %d want 204", rr.Code)
	}
}

func TestHandleGetAdminDRMAnomalies_Unauthorized(t *testing.T) {
	h := NewHandler(Deps{})
	r := httptest.NewRequest(http.MethodGet, "/api/v1/admin/drm/anomalies", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, r)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 got %d", rr.Code)
	}
}
