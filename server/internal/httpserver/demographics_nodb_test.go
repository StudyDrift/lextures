package httpserver

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/lextures/lextures/server/internal/auth"
	"github.com/lextures/lextures/server/internal/config"
)

func demographicsTestToken(t *testing.T, signer *auth.JWTSigner) string {
	t.Helper()
	tok, err := signer.Sign(context.Background(), "00000000-0000-0000-0000-000000000001", "u@test.invalid", "", "", nil)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	return tok
}

func TestDemographicsRoutes_NotFound404(t *testing.T) {
	signer := auth.NewJWTSigner("01234567890123456789012345678901")
	cfg := config.Config{FFDemographics: true}
	h := NewHandler(Deps{Pool: nil, JWTSigner: signer, Config: cfg})
	tok := demographicsTestToken(t, signer)

	cases := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/api/v1/admin/students/00000000-0000-0000-0000-000000000001/demographics"},
		{http.MethodPatch, "/api/v1/admin/students/00000000-0000-0000-0000-000000000001/demographics"},
		{http.MethodGet, "/api/v1/admin/org-units/00000000-0000-0000-0000-000000000002/demographics/report"},
		{http.MethodGet, "/api/v1/admin/org-units/00000000-0000-0000-0000-000000000002/demographics/disaggregated-performance"},
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

func TestDemographicsRoutes_FeatureOff501(t *testing.T) {
	signer := auth.NewJWTSigner("01234567890123456789012345678901")
	cfg := config.Config{FFDemographics: false}
	h := NewHandler(Deps{Pool: nil, JWTSigner: signer, Config: cfg})
	tok := demographicsTestToken(t, signer)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/students/00000000-0000-0000-0000-000000000001/demographics", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusNotImplemented {
		t.Fatalf("expected 501 when feature off, got %d: %s", rr.Code, rr.Body.String())
	}
}
