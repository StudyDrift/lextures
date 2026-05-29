package httpserver

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/lextures/lextures/server/internal/config"
)

func TestReadingPreferences_FeatureFlagOff_Get(t *testing.T) {
	d := Deps{Config: config.Config{FFHighContrastReducedMotion: false}}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/me/reading-preferences", nil)
	rec := httptest.NewRecorder()
	d.handleGetMyReadingPreferences()(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status=%d want 404", rec.Code)
	}
}

func TestReadingPreferences_FeatureFlagOff_Patch(t *testing.T) {
	d := Deps{Config: config.Config{FFHighContrastReducedMotion: false}}
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/me/reading-preferences", nil)
	rec := httptest.NewRecorder()
	d.handlePatchMyReadingPreferences()(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status=%d want 404", rec.Code)
	}
}
