package httpserver

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/lextures/lextures/server/internal/config"
)

func TestGraderAgent_DisabledReturns404(t *testing.T) {
	d := Deps{Config: config.Config{GraderAgentEnabled: false}}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/courses/demo/assignments/00000000-0000-0000-0000-000000000001/grader-agent", nil)
	rec := httptest.NewRecorder()
	d.handleGetGraderAgentConfig()(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status=%d want 404", rec.Code)
	}
}

func TestGraderAgent_DryRunDisabledReturns404(t *testing.T) {
	d := Deps{Config: config.Config{GraderAgentEnabled: false}}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/courses/demo/assignments/00000000-0000-0000-0000-000000000001/grader-agent/dry-run", nil)
	rec := httptest.NewRecorder()
	d.handlePostGraderAgentDryRun()(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status=%d want 404", rec.Code)
	}
}