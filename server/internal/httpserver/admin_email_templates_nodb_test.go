package httpserver

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/lextures/lextures/server/internal/config"
)

func TestAdminEmailTemplates_DisabledReturns404(t *testing.T) {
	d := Deps{Config: config.Config{EmailTemplateEditorEnabled: false, AdminConsoleEnabled: true}}
	rr := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api/v1/admin-console/email-templates", nil)
	d.handleAdminEmailTemplatesList()(rr, r)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestAdminEmailTemplates_UnauthenticatedReturns401(t *testing.T) {
	d := Deps{Config: config.Config{EmailTemplateEditorEnabled: true, AdminConsoleEnabled: true}}
	rr := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api/v1/admin-console/email-templates", nil)
	d.handleAdminEmailTemplatesList()(rr, r)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d", rr.Code)
	}
}

func TestAdminEmailTemplates_MethodNotAllowed(t *testing.T) {
	d := Deps{Config: config.Config{EmailTemplateEditorEnabled: true}}
	rr := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/api/v1/admin-console/email-templates", nil)
	d.handleAdminEmailTemplatesList()(rr, r)
	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status=%d", rr.Code)
	}
}
