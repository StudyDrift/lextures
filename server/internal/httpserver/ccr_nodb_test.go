package httpserver

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/lextures/lextures/server/internal/auth"
	"github.com/lextures/lextures/server/internal/config"
)

func TestCCR_Unauthenticated(t *testing.T) {
	signer := auth.NewJWTSigner("01234567890123456789012345678901")
	cfg := config.Config{FFCoCurricularTranscript: true}
	d := Deps{Pool: nil, JWTSigner: signer, Config: cfg}
	h := NewHandler(d)

	paths := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/api/v1/me/ccr"},
		{http.MethodPost, "/api/v1/me/ccr/generate"},
		{http.MethodGet, "/api/v1/me/ccr/00000000-0000-0000-0000-000000000001/download"},
		{http.MethodPost, "/api/v1/admin/students/00000000-0000-0000-0000-000000000002/ccr/achievements"},
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

func TestCCR_FeatureOff(t *testing.T) {
	signer := auth.NewJWTSigner("01234567890123456789012345678901")
	cfg := config.Config{FFCoCurricularTranscript: false}
	d := Deps{Pool: nil, JWTSigner: signer, Config: cfg}
	h := NewHandler(d)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/verify/test-token", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("want 404 got %d", w.Code)
	}
}

func TestVerifyCCR_PublicWithoutAuth(t *testing.T) {
	signer := auth.NewJWTSigner("01234567890123456789012345678901")
	cfg := config.Config{FFCoCurricularTranscript: true}
	d := Deps{Pool: nil, JWTSigner: signer, Config: cfg}
	h := NewHandler(d)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/verify/missing-token", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("without DB want 500 got %d", w.Code)
	}
}
