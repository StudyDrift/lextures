package httpserver

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/lextures/lextures/server/internal/auth"
	"github.com/lextures/lextures/server/internal/config"
)

func TestPersistentTutor_Unauthenticated(t *testing.T) {
	signer := auth.NewJWTSigner("01234567890123456789012345678901")
	cfg := config.Config{FFPersistentTutor: true}
	d := Deps{Pool: nil, JWTSigner: signer, Config: cfg}
	h := NewHandler(d)

	paths := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/api/v1/courses/C-FAKE/tutor/sessions"},
		{http.MethodPost, "/api/v1/courses/C-FAKE/tutor/sessions"},
		{http.MethodGet, "/api/v1/settings/ai-tutor-opt-out"},
	}
	for _, tc := range paths {
		req := httptest.NewRequest(tc.method, tc.path, nil)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		if w.Code != http.StatusUnauthorized {
			t.Errorf("%s %s: want 401 got %d", tc.method, tc.path, w.Code)
		}
	}
}

func TestAITutorOptOut_AvailableWhenPersistentTutorOff(t *testing.T) {
	signer := auth.NewJWTSigner("01234567890123456789012345678901")
	cfg := config.Config{FFPersistentTutor: false}
	d := Deps{Pool: nil, JWTSigner: signer, Config: cfg}
	h := NewHandler(d)

	for _, method := range []string{http.MethodGet, http.MethodPut} {
		req := httptest.NewRequest(method, "/api/v1/settings/ai-tutor-opt-out", strings.NewReader(`{"aiTutorOptOut":true}`))
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		if w.Code == http.StatusNotFound {
			t.Fatalf("%s: persistent-tutor gate still applied: %s", method, w.Body.String())
		}
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("%s: want 401 got %d body=%s", method, w.Code, w.Body.String())
		}
	}
}

func TestPersistentTutor_FeatureOff(t *testing.T) {
	signer := auth.NewJWTSigner("01234567890123456789012345678901")
	cfg := config.Config{FFPersistentTutor: false}
	d := Deps{Pool: nil, JWTSigner: signer, Config: cfg}
	h := NewHandler(d)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/courses/C-FAKE/tutor/sessions", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("want 404 got %d", w.Code)
	}
}

func TestPersistentTutor_MessageWithoutAIProvider(t *testing.T) {
	h := NewHandler(Deps{Pool: nil, JWTSigner: nil, Config: config.Config{FFPersistentTutor: true}})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/courses/C-FAKE/tutor/sessions/sid/messages", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("want 503 got %d", w.Code)
	}
}
