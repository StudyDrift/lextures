package telemetry

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPackageHelpers_NoopWhenUnset(t *testing.T) {
	defaultMetrics.Store(nil)
	// Must not panic when telemetry was never initialised.
	RecordBusinessEvent("enrollment_created")
	RecordAIProvider("openai", "gpt-4", "ok", 1, 0.01)
	if Default() != nil {
		t.Error("Default() should be nil when unset")
	}
}

func TestPackageHelpers_RecordToDefault(t *testing.T) {
	tel := Init(context.Background(), Config{})
	t.Cleanup(func() { defaultMetrics.Store(nil) })

	RecordBusinessEvent("grade_submitted")
	RecordAIProvider("anthropic", "claude", "ok", 0.5, 0.002)

	rr := httptest.NewRecorder()
	tel.MetricsHandler().ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	body := rr.Body.String()
	if !strings.Contains(body, `lextures_business_events_total{event="grade_submitted"} 1`) {
		t.Error("business event not recorded to default instance")
	}
	if !strings.Contains(body, `provider="anthropic"`) {
		t.Error("AI provider metric not recorded to default instance")
	}
}
