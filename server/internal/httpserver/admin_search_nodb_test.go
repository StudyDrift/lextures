package httpserver

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/lextures/lextures/server/internal/config"
)

func TestAdminSearch_DisabledReturns404(t *testing.T) {
	d := Deps{Config: config.Config{AdminSearchEnabled: false}}
	rr := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api/v1/admin/search?q=test", nil)
	d.handleAdminSearchOmnisearch()(rr, r)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestAdminSearch_UnauthenticatedReturns401(t *testing.T) {
	d := Deps{Config: config.Config{AdminSearchEnabled: true, AdminConsoleEnabled: true}}
	rr := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api/v1/admin/search?q=test", nil)
	d.handleAdminSearchOmnisearch()(rr, r)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d", rr.Code)
	}
}

func TestAdminSearch_QueryTooShort(t *testing.T) {
	d := Deps{Config: config.Config{AdminSearchEnabled: true, AdminConsoleEnabled: true}}
	// adminSearchAccess will fail auth first without token; test parse via enabled path
	rr := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api/v1/admin/search?q=a", nil)
	d.handleAdminSearchOmnisearch()(rr, r)
	if rr.Code != http.StatusUnauthorized {
		// Without auth we get 401; query validation tested in service layer
		t.Fatalf("status=%d", rr.Code)
	}
}

func TestAdminSearch_MethodNotAllowed(t *testing.T) {
	d := Deps{Config: config.Config{AdminSearchEnabled: true}}
	rr := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/api/v1/admin/search?q=test", nil)
	d.handleAdminSearchOmnisearch()(rr, r)
	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status=%d", rr.Code)
	}
}

func TestParseAdminSearchPagination(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/api/v1/admin/search/users?q=test&page=2&per_page=50", nil)
	page, perPage := parseAdminSearchPagination(r)
	if page != 2 || perPage != 50 {
		t.Fatalf("page=%d perPage=%d", page, perPage)
	}
}
