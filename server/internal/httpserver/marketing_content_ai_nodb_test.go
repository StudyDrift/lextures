package httpserver

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/lextures/lextures/server/internal/auth"
	"github.com/lextures/lextures/server/internal/config"
)

func TestGenerateMarketingArticle_FeatureOffReturns404(t *testing.T) {
	h := NewHandler(Deps{Config: config.Config{FFMarketingContent: false}})
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/marketing/articles/generate", strings.NewReader(`{"prompt":"x"}`))
	req.Header.Set("Content-Type", "application/json")
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("got %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestGenerateMarketingArticle_Unauthenticated(t *testing.T) {
	signer := auth.NewJWTSigner("01234567890123456789012345678901")
	h := NewHandler(Deps{JWTSigner: signer, Config: config.Config{FFMarketingContent: true}})
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/marketing/articles/generate", strings.NewReader(`{"prompt":"x"}`))
	req.Header.Set("Content-Type", "application/json")
	h.ServeHTTP(rr, req)
	if rr.Code == http.StatusNotFound {
		t.Fatalf("expected route registered, got 404: %s", rr.Body.String())
	}
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 without auth, got %d: %s", rr.Code, rr.Body.String())
	}
}
