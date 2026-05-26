package httpserver

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/lextures/lextures/server/internal/config"
)

func TestSelfReflection_FeatureDisabled_Returns404(t *testing.T) {
	d := Deps{Config: config.Config{SelfReflectionEnabled: false}}
	h := NewHandler(d)

	for _, path := range []string{
		"/api/v1/me/study-stats",
		"/api/v1/me/study-goal",
		"/api/v1/me/reflection-journal",
		"/api/v1/me/coaching-tips",
	} {
		r := httptest.NewRequest(http.MethodGet, path, nil)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != http.StatusNotFound {
			t.Fatalf("%s: status=%d want 404", path, w.Code)
		}
	}
}

func TestSelfReflection_Unauthenticated_Returns401(t *testing.T) {
	d := Deps{Config: config.Config{SelfReflectionEnabled: true}}
	h := NewHandler(d)
	r := httptest.NewRequest(http.MethodGet, "/api/v1/me/study-stats", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d want 401", w.Code)
	}
}
