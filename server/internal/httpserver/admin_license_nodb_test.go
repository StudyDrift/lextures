package httpserver

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/lextures/lextures/server/internal/config"
)

func TestAdminLicense_DisabledReturns404(t *testing.T) {
	d := Deps{Config: config.Config{SeatManagementEnabled: false, AdminConsoleEnabled: true}}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin-console/license", nil)
	w := httptest.NewRecorder()
	d.handleAdminConsoleLicenseGet()(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("status: %d body %s", w.Code, w.Body.String())
	}
}

func TestAdminLicense_ConsoleRequiresAuth(t *testing.T) {
	d := Deps{Config: config.Config{SeatManagementEnabled: true, AdminConsoleEnabled: true}}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin-console/license", nil)
	w := httptest.NewRecorder()
	d.handleAdminConsoleLicenseGet()(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status: %d", w.Code)
	}
}

func TestAdminLicense_SuperAdminListRequiresAuth(t *testing.T) {
	d := Deps{Config: config.Config{SeatManagementEnabled: true, AdminConsoleEnabled: true}}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/licenses", nil)
	w := httptest.NewRecorder()
	d.handleAdminLicensesList()(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status: %d", w.Code)
	}
}
