package httpserver

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/lextures/lextures/server/internal/auth"
	"github.com/lextures/lextures/server/internal/config"
)

func advisingTestToken(t *testing.T, signer *auth.JWTSigner) string {
	t.Helper()
	tok, err := signer.Sign(context.Background(), "00000000-0000-0000-0000-000000000001", "u@test.invalid", "", "", nil)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	return tok
}

func TestAdvising_Unauthenticated(t *testing.T) {
	signer := auth.NewJWTSigner("01234567890123456789012345678901")
	cfg := config.Config{FFAdvisingIntegration: true}
	d := Deps{Pool: nil, JWTSigner: signer, Config: cfg}
	h := NewHandler(d)

	paths := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/api/v1/me/degree-progress"},
		{http.MethodGet, "/api/v1/me/advising-notes"},
		{http.MethodGet, "/api/v1/me/advising/config"},
		{http.MethodPost, "/api/v1/advisor/students/00000000-0000-0000-0000-000000000002/notes"},
		{http.MethodGet, "/api/v1/advisor/students/00000000-0000-0000-0000-000000000002/notes"},
		{http.MethodGet, "/api/v1/admin/advising/config"},
		{http.MethodPost, "/api/v1/admin/advising/config"},
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

func TestAdvising_FeatureOff(t *testing.T) {
	signer := auth.NewJWTSigner("01234567890123456789012345678901")
	cfg := config.Config{FFAdvisingIntegration: false}
	d := Deps{Pool: nil, JWTSigner: signer, Config: cfg}
	h := NewHandler(d)

	tok := advisingTestToken(t, signer)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/me/advising-notes", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("feature off want 404 got %d", w.Code)
	}
}
