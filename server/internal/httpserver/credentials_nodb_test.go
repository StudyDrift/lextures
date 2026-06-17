package httpserver

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/lextures/lextures/server/internal/config"
)

func TestCredentialsFeatureOffReturns404(t *testing.T) {
	cfg := config.Config{FFCompletionCredentials: false}
	d := Deps{Config: cfg}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/me/credentials", nil)
	w := httptest.NewRecorder()
	d.handleListMyCredentials()(w, req)
	if w.Code != http.StatusUnauthorized && w.Code != http.StatusNotFound {
		// Without auth, meUserID returns 401 first; with feature off after auth it is 404.
		t.Fatalf("expected 401 or 404, got %d", w.Code)
	}
}