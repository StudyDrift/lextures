package httpserver

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/lextures/lextures/server/internal/auth"
	"github.com/lextures/lextures/server/internal/config"
)

func TestBilling_Unauthenticated(t *testing.T) {
	signer := auth.NewJWTSigner("01234567890123456789012345678901")
	cfg := config.Config{FFStripeBilling: true}
	d := Deps{Pool: nil, JWTSigner: signer, Config: cfg}
	h := NewHandler(d)

	paths := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/api/v1/me/entitlements"},
		{http.MethodPost, "/api/v1/billing/checkout"},
		{http.MethodGet, "/api/v1/billing/portal"},
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

func TestBilling_FeatureOff(t *testing.T) {
	// Auth runs before the feature gate; unauthenticated callers get 401 even when billing is disabled.
	signer := auth.NewJWTSigner("01234567890123456789012345678901")
	cfg := config.Config{FFStripeBilling: false}
	d := Deps{Pool: nil, JWTSigner: signer, Config: cfg}
	h := NewHandler(d)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/me/entitlements", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("feature off without auth want 401 got %d", w.Code)
	}
}

func TestStripeWebhook_FeatureOff(t *testing.T) {
	d := Deps{Config: config.Config{FFStripeBilling: false}}
	h := NewHandler(d)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/stripe", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("status=%d want 404", w.Code)
	}
}
