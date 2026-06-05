package httpserver

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/lextures/lextures/server/internal/auth"
	"github.com/lextures/lextures/server/internal/config"
)

func uiModeTestToken(t *testing.T, signer *auth.JWTSigner) string {
	t.Helper()
	tok, err := signer.Sign(context.Background(), "00000000-0000-4000-8000-000000000001", "u@test.invalid", "", "", nil)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	return tok
}

// TestUIMode_PatchAdminUserUIMode_NotFound_WhenFlagOff verifies the 404 guard.
// Auth runs first (requires Pool), so we can only test the unauthenticated path here;
// the authenticated+flag-off=404 case is covered by the E2E tests.
func TestUIMode_PatchAdminUserUIMode_NotFound_WhenFlagOff(t *testing.T) {
	signer := auth.NewJWTSigner("01234567890123456789012345678901")
	cfg := config.Config{FFUiMode: false}
	h := NewHandler(Deps{Pool: nil, JWTSigner: signer, Config: cfg})

	// No auth header — should get 401 regardless of flag state.
	body := []byte(`{"uiMode":"k2"}`)
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/admin/users/00000000-0000-4000-8000-000000000002/ui-mode", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for unauthenticated request, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestUIMode_PatchAdminUserUIMode_Unauthorized(t *testing.T) {
	signer := auth.NewJWTSigner("01234567890123456789012345678901")
	cfg := config.Config{FFUiMode: true}
	h := NewHandler(Deps{Pool: nil, JWTSigner: signer, Config: cfg})

	req := httptest.NewRequest(http.MethodPatch, "/api/v1/admin/users/00000000-0000-4000-8000-000000000002/ui-mode", bytes.NewReader([]byte(`{"uiMode":"k2"}`)))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestUIMode_RouteRegistered(t *testing.T) {
	signer := auth.NewJWTSigner("01234567890123456789012345678901")
	cfg := config.Config{FFUiMode: true}
	h := NewHandler(Deps{Pool: nil, JWTSigner: signer, Config: cfg})
	tok := uiModeTestToken(t, signer)

	req := httptest.NewRequest(http.MethodPatch, "/api/v1/admin/users/00000000-0000-4000-8000-000000000002/ui-mode", bytes.NewReader([]byte(`{"uiMode":"k2"}`)))
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	// Route should be registered (not 404/405). Will get 500 or 403 without DB, both are fine.
	if rr.Code == http.StatusNotFound || rr.Code == http.StatusMethodNotAllowed {
		t.Fatalf("route not registered: %d %s", rr.Code, rr.Body.String())
	}
}
