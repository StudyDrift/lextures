package httpserver

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/lextures/lextures/server/internal/config"
)

func TestScormFeatureDisabled_returns404(t *testing.T) {
	d := Deps{Config: testConfigScormOff()}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/courses/C1/scorm-items/00000000-0000-0000-0000-000000000099", nil)
	rec := httptest.NewRecorder()
	d.handleGetModuleScormByItem()(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status %d want 404", rec.Code)
	}
}

func testConfigScormOff() config.Config {
	return config.Config{ScormIngestionEnabled: false}
}
