package httpserver

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/lextures/lextures/server/internal/config"
)

func TestGraderAgentCancelRun_DisabledReturns404(t *testing.T) {
	d := Deps{Config: config.Config{GraderAgentEnabled: true, GraderAgentCancelRunEnabled: false}}
	runID := uuid.New()
	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/courses/demo/assignments/00000000-0000-0000-0000-000000000001/grader-agent/runs/"+runID.String()+"/cancel",
		nil,
	)
	rec := httptest.NewRecorder()
	d.handlePostGraderAgentCancelRun()(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status=%d want 404", rec.Code)
	}
}

func TestGradingAgentRunStatusCache_ttlAndInvalidate(t *testing.T) {
	runID := uuid.New()
	gradingAgentRunStatusCacheSet(runID, "running")
	if status, ok := gradingAgentRunStatusFromCache(runID); !ok || status != "running" {
		t.Fatalf("cache miss or wrong status: %q ok=%v", status, ok)
	}
	gradingAgentRunStatusCacheInvalidate(runID)
	if _, ok := gradingAgentRunStatusFromCache(runID); ok {
		t.Fatal("expected cache entry to be invalidated")
	}

	gradingAgentRunStatusCache.Store(runID, gradingAgentRunStatusCacheEntry{
		status:    "cancelled",
		expiresAt: time.Now().Add(-time.Second),
	})
	if _, ok := gradingAgentRunStatusFromCache(runID); ok {
		t.Fatal("expected expired cache entry to be ignored")
	}
}
