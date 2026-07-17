package httpserver

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/lextures/lextures/server/internal/config"
)

func TestWalletList_FeatureOff(t *testing.T) {
	h := NewHandler(Deps{Config: config.Config{}})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/me/wallet", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized && rec.Code != http.StatusNotFound {
		// Unauthenticated first; feature gate applies after auth on me routes.
		t.Fatalf("want 401 or 404 got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestWalletPublicShare_FeatureOff(t *testing.T) {
	h := NewHandler(Deps{Config: config.Config{}})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/wallet/s/some-token", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("want 404 got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestWalletPublicShare_FeatureOn_NoPool(t *testing.T) {
	h := NewHandler(Deps{Config: config.Config{FFTranscripts: true}, Pool: nil})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/wallet/s/some-token", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("want 503 got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestWalletExport_Unauthenticated(t *testing.T) {
	h := NewHandler(Deps{Config: config.Config{FFTranscripts: true}})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/me/wallet/export", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("want 401 got %d", rec.Code)
	}
}
