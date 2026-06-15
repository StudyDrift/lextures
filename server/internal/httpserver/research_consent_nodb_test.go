package httpserver

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/lextures/lextures/server/internal/auth"
	"github.com/lextures/lextures/server/internal/config"
)

func consentTestToken(t *testing.T, signer *auth.JWTSigner) string {
	t.Helper()
	tok, err := signer.Sign(context.Background(), "00000000-0000-0000-0000-000000000001", "u@test.invalid", "", "", nil)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	return tok
}

func TestResearchConsent_Unauthenticated(t *testing.T) {
	signer := auth.NewJWTSigner("01234567890123456789012345678901")
	cfg := config.Config{FFResearchConsent: true}
	d := Deps{Pool: nil, JWTSigner: signer, Config: cfg}
	h := NewHandler(d)

	paths := []struct {
		method string
		path   string
	}{
		{http.MethodPost, "/api/v1/admin/consent-studies"},
		{http.MethodGet, "/api/v1/admin/consent-studies"},
		{http.MethodGet, "/api/v1/admin/consent-studies/00000000-0000-0000-0000-000000000002"},
		{http.MethodPatch, "/api/v1/admin/consent-studies/00000000-0000-0000-0000-000000000002"},
		{http.MethodGet, "/api/v1/admin/consent-studies/00000000-0000-0000-0000-000000000002/records"},
		{http.MethodGet, "/api/v1/admin/consent-studies/00000000-0000-0000-0000-000000000002/export"},
		{http.MethodGet, "/api/v1/me/consent-studies"},
		{http.MethodGet, "/api/v1/me/consent-studies/history"},
		{http.MethodPost, "/api/v1/me/consent-studies/00000000-0000-0000-0000-000000000002/respond"},
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

func TestResearchConsent_FeatureOff(t *testing.T) {
	signer := auth.NewJWTSigner("01234567890123456789012345678901")
	cfg := config.Config{FFResearchConsent: false}
	d := Deps{Pool: nil, JWTSigner: signer, Config: cfg}
	h := NewHandler(d)

	tok := consentTestToken(t, signer)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/me/consent-studies", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("feature off want 404 got %d", w.Code)
	}
}
