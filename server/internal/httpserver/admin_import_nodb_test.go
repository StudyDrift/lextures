package httpserver

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/lextures/lextures/server/internal/config"
)

func TestBulkCsvImport_Disabled(t *testing.T) {
	d := Deps{Config: config.Config{BulkCsvImportEnabled: false, AdminConsoleEnabled: true}}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin-console/imports", nil)
	w := httptest.NewRecorder()
	d.handleAdminImportUpload()(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("status: %d body %s", w.Code, w.Body.String())
	}
}

func TestBulkCsvImport_AdminConsoleDisabled(t *testing.T) {
	d := Deps{Config: config.Config{BulkCsvImportEnabled: true, AdminConsoleEnabled: false}}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin-console/imports", nil)
	w := httptest.NewRecorder()
	d.handleAdminImportUpload()(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("status: %d", w.Code)
	}
}

func TestBulkCsvImport_Unauthenticated(t *testing.T) {
	d := Deps{Config: config.Config{BulkCsvImportEnabled: true, AdminConsoleEnabled: true}}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin-console/imports/job-id", nil)
	w := httptest.NewRecorder()
	d.handleAdminImportStatus()(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status: %d", w.Code)
	}
}
