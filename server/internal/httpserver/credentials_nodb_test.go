package httpserver

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/lextures/lextures/server/internal/auth"
	"github.com/lextures/lextures/server/internal/config"
)

func TestCredentials_Unauthenticated(t *testing.T) {
	signer := auth.NewJWTSigner("01234567890123456789012345678901")
	cfg := config.Config{FFCompletionCredentials: true}
	d := Deps{Pool: nil, JWTSigner: signer, Config: cfg}
	h := NewHandler(d)

	paths := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/api/v1/me/credentials"},
		{http.MethodGet, "/api/v1/credentials/00000000-0000-0000-0000-000000000001/linkedin-params"},
		{http.MethodGet, "/api/v1/credentials/00000000-0000-0000-0000-000000000001/badge-export"},
		{http.MethodPost, "/api/v1/credentials/00000000-0000-0000-0000-000000000001/share"},
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

func TestCredentials_FeatureOff(t *testing.T) {
	signer := auth.NewJWTSigner("01234567890123456789012345678901")
	cfg := config.Config{FFCompletionCredentials: false}
	d := Deps{Pool: nil, JWTSigner: signer, Config: cfg}
	h := NewHandler(d)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/credentials/00000000-0000-0000-0000-000000000099/verify", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("want 404 got %d", w.Code)
	}
}

func TestVerifyCredential_PublicWithoutAuth(t *testing.T) {
	signer := auth.NewJWTSigner("01234567890123456789012345678901")
	cfg := config.Config{FFCompletionCredentials: true}
	d := Deps{Pool: nil, JWTSigner: signer, Config: cfg}
	h := NewHandler(d)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/credentials/00000000-0000-0000-0000-000000000099/verify", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("without DB want 500 got %d", w.Code)
	}
}