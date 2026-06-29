package httpserver

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/lextures/lextures/server/internal/config"
)

func TestAdminConsole_DisabledReturns404(t *testing.T) {
	d := Deps{Config: config.Config{AdminConsoleEnabled: false}}
	rr := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api/v1/admin-console/overview", nil)
	d.handleAdminConsoleOverview()(rr, r)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rr.Code)
	}
}

func TestAdminConsole_UnauthenticatedReturns401(t *testing.T) {
	d := Deps{Config: config.Config{AdminConsoleEnabled: true}}
	rr := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api/v1/admin-console/overview", nil)
	d.handleAdminConsoleOverview()(rr, r)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rr.Code)
	}
}

func TestAdminConsole_UsersUnauthenticatedReturns401(t *testing.T) {
	d := Deps{Config: config.Config{AdminConsoleEnabled: true}}
	rr := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api/v1/admin-console/users", nil)
	d.handleAdminConsoleUsers()(rr, r)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rr.Code)
	}
}

func TestAdminConsole_SettingsMethodNotAllowed(t *testing.T) {
	d := Deps{Config: config.Config{AdminConsoleEnabled: true}}
	rr := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/api/v1/admin-console/settings", nil)
	d.handleAdminConsoleSettings()(rr, r)
	if rr.Code != http.StatusUnauthorized && rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want 401 or 405", rr.Code)
	}
}
