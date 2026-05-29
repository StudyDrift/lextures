package httpserver

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/lextures/lextures/server/internal/config"
)

func TestAltTextEnforcement_DisabledReturns404(t *testing.T) {
	d := Deps{Config: config.Config{AltTextEnforcementEnabled: false}}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/courses/demo/accessibility", nil)
	rec := httptest.NewRecorder()
	d.handleGetCourseAccessibility()(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status=%d want 404", rec.Code)
	}
}

func TestAltTextSuggest_DisabledReturns404(t *testing.T) {
	d := Deps{Config: config.Config{AltTextEnforcementEnabled: false}}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/courses/demo/alt-text/suggest", nil)
	rec := httptest.NewRecorder()
	d.handlePostAltTextSuggest()(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status=%d want 404", rec.Code)
	}
}
