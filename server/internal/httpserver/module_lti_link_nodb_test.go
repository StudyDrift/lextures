package httpserver

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/lextures/lextures/server/internal/auth"
	"github.com/lextures/lextures/server/internal/config"
)

// The module LTI page calls these two routes; they used to be unregistered (404).
func TestModuleLTILinkRoutes_Registered(t *testing.T) {
	signer := auth.NewJWTSigner("01234567890123456789012345678901")
	h := NewHandler(Deps{Pool: nil, JWTSigner: signer, Config: config.Config{}})

	cases := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/api/v1/courses/COURSE01/lti-links/00000000-0000-0000-0000-000000000001"},
		{http.MethodPost, "/api/v1/courses/COURSE01/lti-links/00000000-0000-0000-0000-000000000001/embed-ticket"},
	}
	for _, c := range cases {
		t.Run(c.method+" "+c.path, func(t *testing.T) {
			req := httptest.NewRequest(c.method, c.path, nil)
			rr := httptest.NewRecorder()
			h.ServeHTTP(rr, req)
			if rr.Code == http.StatusNotFound || rr.Code == http.StatusMethodNotAllowed {
				t.Fatalf("route not registered: got %d body=%s", rr.Code, rr.Body.String())
			}
			if rr.Code == http.StatusOK {
				t.Fatalf("unauthenticated request should not succeed: body=%s", rr.Body.String())
			}
		})
	}
}

func TestModuleLTIEmbedTicket_LTIDisabled(t *testing.T) {
	signer := auth.NewJWTSigner("01234567890123456789012345678901")
	h := NewHandler(Deps{Pool: nil, JWTSigner: signer, Config: config.Config{}})
	tok := sbgTestToken(t, signer)

	req := httptest.NewRequest(http.MethodPost,
		"/api/v1/courses/COURSE01/lti-links/00000000-0000-0000-0000-000000000001/embed-ticket", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 when LTI is disabled, got %d body=%s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "LTI is not enabled on this server.") {
		t.Fatalf("expected LTI disabled message, body=%s", rr.Body.String())
	}
}

func TestModuleLTIEmbedTicket_UnauthenticatedHidesLTIStatus(t *testing.T) {
	signer := auth.NewJWTSigner("01234567890123456789012345678901")
	h := NewHandler(Deps{Pool: nil, JWTSigner: signer, Config: config.Config{}})

	req := httptest.NewRequest(http.MethodPost,
		"/api/v1/courses/COURSE01/lti-links/00000000-0000-0000-0000-000000000001/embed-ticket", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 before LTI status, got %d body=%s", rr.Code, rr.Body.String())
	}
	if strings.Contains(rr.Body.String(), "LTI is not enabled") {
		t.Fatalf("anonymous response disclosed LTI status: %s", rr.Body.String())
	}
}
