package httpserver

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/lextures/lextures/server/internal/auth"
	"github.com/lextures/lextures/server/internal/config"
)

func TestAdminAISettings_FeatureDisabled404(t *testing.T) {
	h := NewHandler(Deps{Config: config.Config{AiProviderAbstractionEnabled: false}})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/ai-settings", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status: %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestAdminAISettingsTest_FeatureDisabled404(t *testing.T) {
	h := NewHandler(Deps{Config: config.Config{AiProviderAbstractionEnabled: false}})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/ai-settings/test", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status: %d", rec.Code)
	}
}

func TestAdminAISettings_Unauthorized401(t *testing.T) {
	h := NewHandler(Deps{Config: config.Config{AiProviderAbstractionEnabled: true}})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/ai-settings", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status: %d", rec.Code)
	}
}

func TestAdminAISettings_MethodNotAllowed(t *testing.T) {
	s := auth.NewJWTSigner("test-jwt-secret-min-32-chars-xxxxx")
	h := NewHandler(Deps{JWTSigner: s, Config: config.Config{AiProviderAbstractionEnabled: true}})
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/admin/ai-settings", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusMethodNotAllowed && rec.Code != http.StatusUnauthorized {
		t.Fatalf("status: %d", rec.Code)
	}
}