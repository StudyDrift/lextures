package telemetry

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestMetricsHandler_Exposition(t *testing.T) {
	m := NewMetrics()
	m.SetBuildInfo("1.2.3", "test")
	m.ObserveHTTP(http.MethodGet, "/api/v1/courses", "2xx", 0.012)
	m.ObserveAIProvider("openai", "gpt-4", "ok", 1.5, 0.002)
	m.IncBusinessEvent("enrollment_created")

	rr := httptest.NewRecorder()
	m.Handler().ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("status %d", rr.Code)
	}
	body := rr.Body.String()
	// Plan 17.7 AC-1: exposition includes http request total and duration metrics.
	for _, want := range []string{
		"lextures_http_requests_total",
		"lextures_http_request_duration_seconds",
		`lextures_build_info{deploy_color="stable",env="test",version="1.2.3"} 1`,
		"lextures_ai_provider_calls_total",
		"lextures_ai_estimated_cost_dollars_total",
		`lextures_business_events_total{event="enrollment_created"} 1`,
		// Standard Go runtime collector is registered.
		"go_goroutines",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("exposition missing %q", want)
		}
	}
}

func TestMetrics_DeployColorLabel(t *testing.T) {
	m := NewMetrics("green")
	m.SetBuildInfo("abc123", "production")
	rr := httptest.NewRecorder()
	m.Handler().ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	if !strings.Contains(rr.Body.String(), `deploy_color="green"`) {
		t.Error("expected deploy_color=green on metrics series")
	}
}

func TestMetrics_BuildInfoDefaultsVersion(t *testing.T) {
	m := NewMetrics()
	m.SetBuildInfo("", "prod")
	rr := httptest.NewRecorder()
	m.Handler().ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	if !strings.Contains(rr.Body.String(), `version="dev"`) {
		t.Error("empty version should default to dev")
	}
}

func TestMetrics_AIProviderUnknownLabels(t *testing.T) {
	m := NewMetrics()
	m.ObserveAIProvider("", "", "error", 0.1, 0)
	rr := httptest.NewRecorder()
	m.Handler().ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	body := rr.Body.String()
	if !strings.Contains(body, `provider="unknown"`) || !strings.Contains(body, `model="unknown"`) {
		t.Error("blank provider/model should fall back to unknown")
	}
}

func TestMetrics_IncBusinessEventEmptyNoop(t *testing.T) {
	m := NewMetrics()
	m.IncBusinessEvent("")
	rr := httptest.NewRecorder()
	m.Handler().ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	if strings.Contains(rr.Body.String(), "business_events_total") {
		t.Error("empty event should not create a series")
	}
}

func TestMetrics_SetBannerActive(t *testing.T) {
	m := NewMetrics()
	m.SetBannerActive("global", "warning")
	rr := httptest.NewRecorder()
	m.Handler().ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	body := rr.Body.String()
	if !strings.Contains(body, `lextures_banner_active{scope="global",severity="warning"} 1`) {
		t.Errorf("expected active global warning gauge, got: %s", body)
	}
}
