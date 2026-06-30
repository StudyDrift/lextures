package httpserver

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/lextures/lextures/server/internal/config"
)

func TestCustomFields_Disabled(t *testing.T) {
	d := Deps{Config: config.Config{CustomFieldsEnabled: false, AdminConsoleEnabled: true}}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin-console/custom-fields?entity_type=user", nil)
	w := httptest.NewRecorder()
	d.handleAdminCustomFieldsList()(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("status: %d body %s", w.Code, w.Body.String())
	}
}

func TestCustomFields_AdminConsoleDisabled(t *testing.T) {
	d := Deps{Config: config.Config{CustomFieldsEnabled: true, AdminConsoleEnabled: false}}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin-console/custom-fields?entity_type=user", nil)
	w := httptest.NewRecorder()
	d.handleAdminCustomFieldsList()(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("status: %d", w.Code)
	}
}

func TestCustomFields_Unauthenticated(t *testing.T) {
	d := Deps{Config: config.Config{CustomFieldsEnabled: true, AdminConsoleEnabled: true}}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin-console/custom-fields", nil)
	w := httptest.NewRecorder()
	d.handleAdminCustomFieldsCreate()(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status: %d", w.Code)
	}
}

func TestWantsInclude(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/me?include=custom_fields,profile", nil)
	if !wantsInclude(req, "custom_fields") {
		t.Fatal("expected custom_fields include")
	}
	if wantsInclude(req, "other") {
		t.Fatal("did not expect other include")
	}
}
