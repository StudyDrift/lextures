package httpserver

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/lextures/lextures/server/internal/config"
)

func TestGraderAgentReviewInbox_DisabledReturns404(t *testing.T) {
	d := Deps{Config: config.Config{GraderAgentEnabled: false}}
	item := "00000000-0000-0000-0000-000000000001"
	tests := []struct {
		method string
		path   string
		run    func(http.ResponseWriter, *http.Request)
	}{
		{http.MethodGet, "/api/v1/courses/demo/assignments/" + item + "/grader-agent/runs", d.handleListGraderAgentRuns()},
		{http.MethodGet, "/api/v1/courses/demo/assignments/" + item + "/grader-agent/review-queue", d.handleGetGraderAgentReviewQueue()},
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