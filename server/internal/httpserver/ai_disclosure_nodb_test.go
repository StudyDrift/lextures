package httpserver

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/lextures/lextures/server/internal/auth"
	"github.com/lextures/lextures/server/internal/config"
)

func TestPublicAIDisclosure_OK(t *testing.T) {
	h := NewHandler(Deps{Config: config.Config{AiDisclosureEnabled: true}})
	r := httptest.NewRequest(http.MethodGet, "/api/v1/public/ai-disclosure", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/json; charset=utf-8" {
		t.Fatalf("content-type=%q", ct)
	}
}

func TestAIOptOut_Unauthenticated(t *testing.T) {
	h := NewHandler(Deps{Config: config.Config{AiDisclosureEnabled: true}})
	r := httptest.NewRequest(http.MethodGet, "/api/v1/settings/ai-opt-out", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d", w.Code)
	}
}

func TestAIOptOut_DisabledModule(t *testing.T) {
	h := NewHandler(Deps{JWTSigner: auth.NewJWTSigner("test-jwt-secret-min-32-chars-xxxxx"), Config: config.Config{AiDisclosureEnabled: false}})
	r := httptest.NewRequest(http.MethodGet, "/api/v1/settings/ai-opt-out", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusNotFound {
		t.Fatalf("status=%d", w.Code)
	}
}

func TestAdminAIConfig_Unauthenticated(t *testing.T) {
	h := NewHandler(Deps{Config: config.Config{AiDisclosureEnabled: true}})
	r := httptest.NewRequest(http.MethodGet, "/api/v1/admin/ai-config", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d", w.Code)
	}
}

func TestAIFeatureAck_InvalidBody(t *testing.T) {
	s := auth.NewJWTSigner("test-jwt-secret-min-32-chars-xxxxx")
	h := NewHandler(Deps{JWTSigner: s, Config: config.Config{AiDisclosureEnabled: true}})
	r := httptest.NewRequest(http.MethodPost, "/api/v1/settings/ai-disclosure/acknowledgements", bytes.NewReader([]byte(`{}`)))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusUnauthorized && w.Code != http.StatusBadRequest {
		t.Fatalf("status=%d", w.Code)
	}
}
