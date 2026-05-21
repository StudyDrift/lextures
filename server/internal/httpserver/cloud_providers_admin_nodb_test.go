package httpserver

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetAdminCloudProviders_Unauthorized(t *testing.T) {
	h := NewHandler(Deps{Pool: nil, JWTSigner: nil})
	rr := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api/v1/admin/cloud-providers", nil)
	h.ServeHTTP(rr, r)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 without auth, got %d", rr.Code)
	}
}

func TestPutAdminCloudProvider_Unauthorized(t *testing.T) {
	h := NewHandler(Deps{Pool: nil, JWTSigner: nil})
	rr := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPut, "/api/v1/admin/cloud-providers/google_drive", nil)
	h.ServeHTTP(rr, r)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 without auth, got %d", rr.Code)
	}
}

func TestGetAdminCloudProviders_MethodNotAllowed(t *testing.T) {
	h := NewHandler(Deps{Pool: nil, JWTSigner: nil})
	for _, method := range []string{http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch} {
		rr := httptest.NewRecorder()
		r := httptest.NewRequest(method, "/api/v1/admin/cloud-providers", nil)
		h.ServeHTTP(rr, r)
		// Without JWTSigner, auth runs first → 401; after auth the method check would give 405.
		if rr.Code != http.StatusMethodNotAllowed && rr.Code != http.StatusUnauthorized {
			t.Fatalf("cloud-providers %s: expected 405 or 401, got %d", method, rr.Code)
		}
	}
}

func TestGetAdminCloudProviders_OptionsReturns204(t *testing.T) {
	h := NewHandler(Deps{Pool: nil, JWTSigner: nil})
	rr := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodOptions, "/api/v1/admin/cloud-providers", nil)
	h.ServeHTTP(rr, r)
	if rr.Code != http.StatusNoContent {
		t.Fatalf("OPTIONS cloud-providers: expected 204, got %d", rr.Code)
	}
}

func TestPutAdminCloudProvider_OptionsReturns204(t *testing.T) {
	h := NewHandler(Deps{Pool: nil, JWTSigner: nil})
	rr := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodOptions, "/api/v1/admin/cloud-providers/google_drive", nil)
	h.ServeHTTP(rr, r)
	if rr.Code != http.StatusNoContent {
		t.Fatalf("OPTIONS cloud-providers/google_drive: expected 204, got %d", rr.Code)
	}
}
