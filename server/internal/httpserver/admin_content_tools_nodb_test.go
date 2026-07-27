package httpserver

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/lextures/lextures/server/internal/config"
)

func TestAdminContentToolsRoutesRequireAuth(t *testing.T) {
	h := NewHandler(Deps{Config: config.Config{}, Pool: nil})
	paths := []string{
		"/api/v1/admin/content-tools/versions",
		"/api/v1/admin/content-tools/quarantine",
	}
	for _, p := range paths {
		rr := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, p, nil)
		h.ServeHTTP(rr, req)
		if rr.Code == http.StatusOK {
			t.Fatalf("%s: expected auth failure, got %d", p, rr.Code)
		}
	}
}
