package httpserver

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/lextures/lextures/server/internal/auth"
	"github.com/lextures/lextures/server/internal/config"
)

func TestPostSettingsAccountGenerateAvatar_Not404NoRoute(t *testing.T) {
	signer := auth.NewJWTSigner("01234567890123456789012345678901")
	h := NewHandler(Deps{Pool: nil, JWTSigner: signer, Config: config.Config{}})
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/settings/account/generate-avatar", strings.NewReader(`{"prompt":"a simple portrait"}`))
	req.Header.Set("Content-Type", "application/json")
	tok, err := signer.Sign(context.Background(), "00000000-0000-0000-0000-000000000001", "u@test.com", "", "", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+tok)
	h.ServeHTTP(rr, req)
	if rr.Code == http.StatusNotFound {
		t.Fatalf("expected handler to be registered, got 404: %s", rr.Body.String())
	}
	if strings.Contains(rr.Body.String(), "No HTTP route is registered") {
		t.Fatalf("missing-route error: %s", rr.Body.String())
	}
	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestPostSettingsAccountGenerateAvatar_UnauthenticatedIsNotMissingRoute(t *testing.T) {
	signer := auth.NewJWTSigner("01234567890123456789012345678901")
	h := NewHandler(Deps{Pool: nil, JWTSigner: signer, Config: config.Config{}})
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/settings/account/generate-avatar", strings.NewReader(`{"prompt":"portrait"}`))
	req.Header.Set("Content-Type", "application/json")
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	if strings.Contains(rr.Body.String(), "No HTTP route is registered") {
		t.Fatalf("missing-route error: %s", rr.Body.String())
	}
}
