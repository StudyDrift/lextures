package httpserver

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/lextures/lextures/server/internal/auth"
	"github.com/lextures/lextures/server/internal/config"
)

func TestMeProfileDepthRoutes_Registered(t *testing.T) {
	signer := auth.NewJWTSigner("01234567890123456789012345678901")
	cfg := config.Config{FFDemographics: true, CustomFieldsEnabled: true}
	h := NewHandler(Deps{Pool: nil, JWTSigner: signer, Config: cfg})
	tok := demographicsTestToken(t, signer)

	cases := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/api/v1/me/profile-fields"},
		{http.MethodPatch, "/api/v1/me/profile-fields"},
		{http.MethodGet, "/api/v1/me/demographics"},
		{http.MethodPatch, "/api/v1/me/demographics"},
	}

	for _, c := range cases {
		t.Run(c.method+" "+c.path, func(t *testing.T) {
			req := httptest.NewRequest(c.method, c.path, nil)
			req.Header.Set("Authorization", "Bearer "+tok)
			rr := httptest.NewRecorder()
			h.ServeHTTP(rr, req)
			if rr.Code == http.StatusNotFound {
				t.Fatalf("expected route to be registered, got 404 for %s %s: %s",
					c.method, c.path, rr.Body.String())
			}
		})
	}
}

func TestMeProfileFields_Unauthenticated(t *testing.T) {
	h := NewHandler(Deps{JWTSigner: auth.NewJWTSigner("01234567890123456789012345678901"), Config: config.Config{}})
	rr := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api/v1/me/profile-fields", nil)
	h.ServeHTTP(rr, r)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("GET profile-fields unauthenticated: status=%d want 401", rr.Code)
	}
}