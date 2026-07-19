package httpserver

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/lextures/lextures/server/internal/config"
)

func TestGraderAgentSuggestMode_DisabledReturns404(t *testing.T) {
	d := Deps{Config: config.Config{GraderAgentEnabled: false}}
	item := "00000000-0000-0000-0000-000000000001"
	req := httptest.NewRequest(http.MethodPost, "/api/v1/courses/demo/assignments/"+item+"/grader-agent/review/bulk", nil)
	rec := httptest.NewRecorder()
	d.handlePostGraderAgentReviewBulk()(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}