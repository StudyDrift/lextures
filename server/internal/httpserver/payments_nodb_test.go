package httpserver

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/lextures/lextures/server/internal/auth"
	"github.com/lextures/lextures/server/internal/config"
)

func TestPayments_Unauthenticated(t *testing.T) {
	signer := auth.NewJWTSigner("01234567890123456789012345678901")
	cfg := config.Config{FFPaymentsEnabled: true}
	d := Deps{Pool: nil, JWTSigner: signer, Config: cfg}
	h := NewHandler(d)

	paths := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/api/v1/me/transactions"},
		{http.MethodPost, "/api/v1/checkout"},
		{http.MethodPost, "/api/v1/admin/transactions/00000000-0000-4000-8000-000000000001/refund"},
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

func TestPayPalWebhook_FeatureOff(t *testing.T) {
	d := Deps{Config: config.Config{FFPaymentsEnabled: false}}
	h := NewHandler(d)
	req := httptest.NewRequest(http.MethodPost, "/webhooks/paypal", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("status=%d want 404", w.Code)
	}
}

func TestStripeWebhook_NoDatabase(t *testing.T) {
	d := Deps{Config: config.Config{
		FFPaymentsEnabled:   true,
		StripeWebhookSecret: "whsec_test",
	}}
	h := NewHandler(d)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/stripe", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("status=%d want 503", w.Code)
	}
}
