package httpserver

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/lextures/lextures/server/internal/auth"
)

func TestAccessKeysRoutes_NoDB(t *testing.T) {
	t.Parallel()
	jwtSecret := "01234567890123456789012345678901"
	d := Deps{JWTSigner: auth.NewJWTSigner(jwtSecret)}
	h := NewHandler(d)
	routes := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/api/v1/me/access-keys/scopes"},
		{http.MethodGet, "/api/v1/me/access-keys"},
		{http.MethodPost, "/api/v1/me/access-keys"},
		{http.MethodDelete, "/api/v1/me/access-keys/00000000-0000-0000-0000-000000000001"},
		{http.MethodGet, "/api/v1/me/integrations/mcp"},
	}
	for _, rt := range routes {
		req := httptest.NewRequest(rt.method, rt.path, nil)
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, req)
		if rr.Code != http.StatusUnauthorized {
			t.Fatalf("%s %s: expected 401, got %d", rt.method, rt.path, rr.Code)
		}
	}
}
