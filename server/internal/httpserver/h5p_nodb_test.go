package httpserver

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/lextures/lextures/server/internal/config"
)

func TestH5PFeatureDisabled_returns404(t *testing.T) {
	d := Deps{Config: testConfigH5POff()}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/courses/C1/h5p/00000000-0000-0000-0000-000000000001", nil)
	rec := httptest.NewRecorder()
	d.handleGetH5PPackage()(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status %d want 404", rec.Code)
	}
}

func TestXAPIFeatureDisabled_returns404(t *testing.T) {
	d := Deps{Config: testConfigH5POff()}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/xapi/statements", nil)
	rec := httptest.NewRecorder()
	d.handlePostXAPIStatements()(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status %d want 404", rec.Code)
	}
}

func testConfigH5POff() config.Config {
	return config.Config{H5PEnabled: false}
}
