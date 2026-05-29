package httpserver

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestReadingPreferences_GetUnauthorized(t *testing.T) {
	d := Deps{}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/me/reading-preferences", nil)
	rec := httptest.NewRecorder()
	d.handleGetMyReadingPreferences()(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d want 401", rec.Code)
	}
}

func TestReadingPreferences_PatchUnauthorized(t *testing.T) {
	d := Deps{}
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/me/reading-preferences",
		strings.NewReader(`{"fontFace":"atkinson"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	d.handlePatchMyReadingPreferences()(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d want 401", rec.Code)
	}
}
