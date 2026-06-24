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

func TestGraderAgent_TemplateDisabledReturns404(t *testing.T) {
	d := Deps{Config: config.Config{GraderAgentEnabled: false}}
	tests := []struct {
		method string
		path   string
		run    func(http.ResponseWriter, *http.Request)
	}{
		{http.MethodGet, "/api/v1/courses/demo/grader-agent-templates", d.handleListGraderAgentTemplates()},
		{http.MethodPost, "/api/v1/courses/demo/grader-agent-templates", d.handlePostGraderAgentTemplate()},
		{http.MethodGet, "/api/v1/courses/demo/grader-agent-templates/00000000-0000-0000-0000-000000000001", d.handleGetGraderAgentTemplate()},
		{http.MethodPut, "/api/v1/courses/demo/grader-agent-templates/00000000-0000-0000-0000-000000000001", d.handlePutGraderAgentTemplate()},
		{http.MethodDelete, "/api/v1/courses/demo/grader-agent-templates/00000000-0000-0000-0000-000000000001", d.handleDeleteGraderAgentTemplate()},
	}
	for _, tc := range tests {
		req := httptest.NewRequest(tc.method, tc.path, nil)
		rec := httptest.NewRecorder()
		tc.run(rec, req)
		if rec.Code != http.StatusNotFound {
			t.Fatalf("%s %s status=%d want 404", tc.method, tc.path, rec.Code)
		}
	}
}