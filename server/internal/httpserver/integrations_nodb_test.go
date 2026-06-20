package httpserver

import (
	"net/http"
	"net/http/httptest"
	"testing"

	integrations "github.com/lextures/lextures/server/internal/service/integrations"
)

func TestIntegrations_NotImplementedWhenServiceNil(t *testing.T) {
	h := NewHandler(Deps{}) // no Integrations service wired
	for _, path := range []string{
		"/api/v1/integrations",
		"/api/v1/integrations/" + "11111111-1111-1111-1111-111111111111" + "/sync-status",
	} {
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, path, nil))
		if rr.Code != http.StatusNotImplemented {
			t.Errorf("GET %s: expected 501, got %d", path, rr.Code)
		}
	}
}

func TestIntegrations_OptionsReturns204(t *testing.T) {
	h := NewHandler(Deps{Integrations: integrations.NewService(nil, "https://x", []byte("s"))})
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodOptions, "/api/v1/integrations", nil))
	if rr.Code != http.StatusNoContent {
		t.Errorf("OPTIONS integrations: expected 204, got %d", rr.Code)
	}
}

func TestIntegrations_ListUnauthorized(t *testing.T) {
	// Service wired but no JWT signer => admin auth fails with 401.
	h := NewHandler(Deps{Integrations: integrations.NewService(nil, "https://x", []byte("s"))})
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/api/v1/integrations", nil))
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 without auth, got %d", rr.Code)
	}
}

func TestIntegrations_ConnectUnauthorized(t *testing.T) {
	h := NewHandler(Deps{Integrations: integrations.NewService(nil, "https://x", []byte("s"))})
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/integrations/oauth/google_classroom/connect", nil))
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("connect without auth: expected 401, got %d", rr.Code)
	}
}

func TestIntegrations_CallbackInvalidStateRedirects(t *testing.T) {
	// The callback is unauthenticated (state-carried). A bad state must not 500;
	// it redirects back to the admin page with an error param.
	h := NewHandler(Deps{Integrations: integrations.NewService(nil, "https://x", []byte("s"))})
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/integrations/oauth/google_classroom/callback?code=c&state=bad", nil))
	if rr.Code != http.StatusFound {
		t.Fatalf("callback invalid state: expected 302, got %d", rr.Code)
	}
	if loc := rr.Header().Get("Location"); loc == "" || loc[:19] != "/admin/integrations" {
		t.Errorf("unexpected redirect location: %q", loc)
	}
}

func TestIntegrations_CallbackProviderErrorRedirects(t *testing.T) {
	h := NewHandler(Deps{Integrations: integrations.NewService(nil, "https://x", []byte("s"))})
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/integrations/oauth/google_classroom/callback?error=access_denied", nil))
	if rr.Code != http.StatusFound {
		t.Fatalf("callback provider error: expected 302, got %d", rr.Code)
	}
}
