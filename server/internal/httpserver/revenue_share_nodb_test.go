package httpserver

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/lextures/lextures/server/internal/auth"
	"github.com/lextures/lextures/server/internal/config"
)

func TestRevenueShare_Unauthenticated(t *testing.T) {
	signer := auth.NewJWTSigner("01234567890123456789012345678901")
	cfg := config.Config{FFRevenueShare: true, FFStripeBilling: true}
	d := Deps{Pool: nil, JWTSigner: signer, Config: cfg}
	h := NewHandler(d)

	paths := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/api/v1/creator/earnings"},
		{http.MethodGet, "/api/v1/creator/earnings/ledger"},
		{http.MethodPost, "/api/v1/creator/affiliate-codes"},
		{http.MethodGet, "/api/v1/creator/affiliate-codes"},
	}
	for _, tc := range paths {
		req := httptest.NewRequest(tc.method, tc.path, nil)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		if w.Code != http.StatusUnauthorized {
			t.Errorf("%s %s: want 401 got %d", tc.method, tc.path, w.Code)
		}
	}
}

func TestRevenueShare_FeatureOff(t *testing.T) {
	signer := auth.NewJWTSigner("01234567890123456789012345678901")
	cfg := config.Config{FFRevenueShare: false, FFStripeBilling: true}
	d := Deps{Pool: nil, JWTSigner: signer, Config: cfg}
	h := NewHandler(d)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/creator/earnings", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("feature off without auth want 401 got %d", w.Code)
	}
}

func TestAffiliateTrackClick_FeatureOff(t *testing.T) {
	d := Deps{Config: config.Config{FFRevenueShare: false}}
	h := NewHandler(d)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/affiliate/track-click?code=abc", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("status=%d want 404", w.Code)
	}
}
