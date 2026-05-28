package httpserver

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/lextures/lextures/server/internal/config"
)

func TestRedactionStatus_Unauthenticated_Returns401(t *testing.T) {
	d := Deps{Config: config.Config{AppEnv: "local"}}
	h := NewHandler(d)
	r := httptest.NewRequest(http.MethodGet, "/api/v1/internal/ops/redaction-status", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d want 401", w.Code)
	}
}
