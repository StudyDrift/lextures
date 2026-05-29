package httpserver

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/lextures/lextures/server/internal/config"
)

func TestReadingPreferences_Unauthenticated_Get(t *testing.T) {
	d := Deps{Config: config.Config{}, JWTSigner: nil}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/me/reading-preferences", nil)
	rec := httptest.NewRecorder()
	d.handleGetMyReadingPreferences()(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d want 401", rec.Code)
	}
}

func TestReadingPreferences_Unauthenticated_Patch(t *testing.T) {
	d := Deps{Config: config.Config{}, JWTSigner: nil}
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/me/reading-preferences", nil)
	rec := httptest.NewRecorder()
	d.handlePatchMyReadingPreferences()(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d want 401", rec.Code)
	}
}
