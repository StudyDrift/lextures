package httpserver

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/lextures/lextures/server/internal/service/storagequota"
)

// noopQuotaService is a storagequota.Service with nil pool — used so the feature-enabled
// path is taken without needing a real database (auth is verified before any DB call).
var noopQuotaService = &storagequota.Service{Pool: nil}

// ---------------------------------------------------------------------------
// 501 when StorageQuota service is nil (feature disabled)
// ---------------------------------------------------------------------------

func TestStorageUsage_NoService_Returns501(t *testing.T) {
	h := NewHandler(Deps{Pool: nil, JWTSigner: nil, StorageQuota: nil})
	rr := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api/v1/courses/cs101/storage-usage", nil)
	h.ServeHTTP(rr, r)
	if rr.Code != http.StatusNotImplemented {
		t.Fatalf("expected 501 got %d", rr.Code)
	}
}

func TestAdminStorageQuotasList_NoService_Returns501(t *testing.T) {
	h := NewHandler(Deps{Pool: nil, JWTSigner: nil, StorageQuota: nil})
	rr := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api/v1/admin/storage-quotas", nil)
	h.ServeHTTP(rr, r)
	if rr.Code != http.StatusNotImplemented {
		t.Fatalf("expected 501 got %d", rr.Code)
	}
}

func TestAdminStorageQuotasPut_NoService_Returns501(t *testing.T) {
	h := NewHandler(Deps{Pool: nil, JWTSigner: nil, StorageQuota: nil})
	rr := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPut,
		"/api/v1/admin/storage-quotas/course/00000000-0000-0000-0000-000000000001", nil)
	h.ServeHTTP(rr, r)
	if rr.Code != http.StatusNotImplemented {
		t.Fatalf("expected 501 got %d", rr.Code)
	}
}

func TestAdminStorageQuotasReconcile_NoService_Returns501(t *testing.T) {
	h := NewHandler(Deps{Pool: nil, JWTSigner: nil, StorageQuota: nil})
	rr := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/api/v1/admin/storage-quotas/reconcile", nil)
	h.ServeHTTP(rr, r)
	if rr.Code != http.StatusNotImplemented {
		t.Fatalf("expected 501 got %d", rr.Code)
	}
}

// ---------------------------------------------------------------------------
// Auth guard: 401 when no JWT is provided (service is enabled)
// ---------------------------------------------------------------------------

func TestStorageUsage_NoJWT_Returns401(t *testing.T) {
	h := NewHandler(Deps{Pool: nil, JWTSigner: nil, StorageQuota: noopQuotaService})
	rr := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api/v1/courses/cs101/storage-usage", nil)
	h.ServeHTTP(rr, r)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 got %d", rr.Code)
	}
}

func TestAdminStorageQuotasList_NoJWT_Returns401(t *testing.T) {
	h := NewHandler(Deps{Pool: nil, JWTSigner: nil, StorageQuota: noopQuotaService})
	rr := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api/v1/admin/storage-quotas", nil)
	h.ServeHTTP(rr, r)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 got %d", rr.Code)
	}
}

func TestAdminStorageQuotasPut_NoJWT_Returns401(t *testing.T) {
	h := NewHandler(Deps{Pool: nil, JWTSigner: nil, StorageQuota: noopQuotaService})
	rr := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPut,
		"/api/v1/admin/storage-quotas/course/00000000-0000-0000-0000-000000000001", nil)
	h.ServeHTTP(rr, r)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 got %d", rr.Code)
	}
}

func TestAdminStorageQuotasReconcile_NoJWT_Returns401(t *testing.T) {
	h := NewHandler(Deps{Pool: nil, JWTSigner: nil, StorageQuota: noopQuotaService})
	rr := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/api/v1/admin/storage-quotas/reconcile", nil)
	h.ServeHTTP(rr, r)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 got %d", rr.Code)
	}
}
