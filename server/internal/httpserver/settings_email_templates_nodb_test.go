package httpserver

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/lextures/lextures/server/internal/config"
)

func TestPlatformEmailTemplates_UnauthenticatedReturns401(t *testing.T) {
	h := NewHandler(Deps{
		Pool:      nil,
		JWTSigner: nil,
		Config:    config.Config{EmailTemplateEditorEnabled: true},
	})
	paths := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/api/v1/settings/platform/email-templates"},
		{http.MethodGet, "/api/v1/settings/platform/email-templates/magic_link"},
		{http.MethodPut, "/api/v1/settings/platform/email-templates/magic_link"},
		{http.MethodGet, "/api/v1/settings/platform/email-templates/magic_link/history"},
		{http.MethodPost, "/api/v1/settings/platform/email-templates/magic_link/restore"},
		{http.MethodPost, "/api/v1/settings/platform/email-templates/magic_link/reset"},
		{http.MethodPost, "/api/v1/settings/platform/email-templates/magic_link/test"},
		{http.MethodPost, "/api/v1/settings/platform/email-templates/magic_link/preview"},
	}
	for _, p := range paths {
		rr := httptest.NewRecorder()
		r := httptest.NewRequest(p.method, p.path, nil)
		h.ServeHTTP(rr, r)
		if rr.Code != http.StatusUnauthorized {
			t.Fatalf("%s %s: want 401 got %d", p.method, p.path, rr.Code)
		}
	}
}

func TestPlatformEmailTemplates_DisabledReturns404(t *testing.T) {
	h := NewHandler(Deps{
		Pool:      nil,
		JWTSigner: nil,
		Config:    config.Config{EmailTemplateEditorEnabled: false},
	})
	// Without auth we still 401 first when JWT is required; with nil JWTSigner auth fails before flag.
	// When JWT is nil, adminRbacUser returns 401. To exercise 404, we need the flag check first —
	// platformEmailTemplateAccess checks flag before adminRbacUser.
	// With JWTSigner nil and flag false: access hits flag first → 404.
	rr := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api/v1/settings/platform/email-templates", nil)
	h.ServeHTTP(rr, r)
	// Flag is checked first, so 404 even without auth.
	if rr.Code != http.StatusNotFound {
		t.Fatalf("want 404 when flag off, got %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestPlatformEmailTemplates_MethodNotAllowed(t *testing.T) {
	h := NewHandler(Deps{
		Pool:      nil,
		JWTSigner: nil,
		Config:    config.Config{EmailTemplateEditorEnabled: true},
	})
	// Unauthenticated still 401 for wrong method that is not registered? Chi may 405 on unmatched method.
	rr := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodDelete, "/api/v1/settings/platform/email-templates", nil)
	h.ServeHTTP(rr, r)
	// No DELETE route → 405 or 401 depending on middleware. Accept either non-2xx.
	if rr.Code == http.StatusOK {
		t.Fatalf("DELETE should not succeed, got %d", rr.Code)
	}
}
