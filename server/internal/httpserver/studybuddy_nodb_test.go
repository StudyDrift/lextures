package httpserver

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/lextures/lextures/server/internal/auth"
	"github.com/lextures/lextures/server/internal/config"
)

func TestStudyBuddy_Unauthenticated(t *testing.T) {
	signer := auth.NewJWTSigner("01234567890123456789012345678901")
	cfg := config.Config{FFAIStudyBuddy: true}
	d := Deps{Pool: nil, JWTSigner: signer, Config: cfg}
	h := NewHandler(d)

	paths := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/api/v1/courses/C-FAKE/study-buddy/memory"},
		{http.MethodDelete, "/api/v1/courses/C-FAKE/study-buddy/memory"},
		{http.MethodGet, "/api/v1/courses/C-FAKE/study-buddy/prompts"},
	}
	for _, tc := range paths {
		req := httptest.NewRequest(tc.method, tc.path, nil)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		if w.Code != http.StatusUnauthorized {
			t.Errorf("%s %s: want 401 got %d", tc.method, tc.path, w.Code)
		}
	}

	// POST checks AI provider before auth (same as tutor message).
	req := httptest.NewRequest(http.MethodPost, "/api/v1/courses/C-FAKE/study-buddy/message", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("POST message without AI: want 503 got %d", w.Code)
	}
}

func TestStudyBuddy_FeatureOff(t *testing.T) {
	signer := auth.NewJWTSigner("01234567890123456789012345678901")
	cfg := config.Config{FFAIStudyBuddy: false}
	d := Deps{Pool: nil, JWTSigner: signer, Config: cfg}
	h := NewHandler(d)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/courses/C-FAKE/study-buddy/prompts", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("want 404 got %d", w.Code)
	}
}

func TestStudyBuddy_MessageWithoutAIProvider(t *testing.T) {
	h := NewHandler(Deps{Pool: nil, JWTSigner: nil, Config: config.Config{FFAIStudyBuddy: true}})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/courses/C-FAKE/study-buddy/message", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("want 503 got %d", w.Code)
	}
}
