package httpserver

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/lextures/lextures/server/internal/config"
)

func TestGetCourseEvents_featureDisabled_returns404(t *testing.T) {
	d := Deps{Config: config.Config{XAPIEmissionEnabled: false}}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/courses/demo/events", nil)
	rec := httptest.NewRecorder()
	d.handleGetCourseEvents()(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
}

func TestGetAdminLRSConfig_unauthenticated_returns401(t *testing.T) {
	d := Deps{Config: config.Config{XAPIEmissionEnabled: true}}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/lrs-config", nil)
	rec := httptest.NewRecorder()
	d.handleGetAdminLRSConfig()(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
}
