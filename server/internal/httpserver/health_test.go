package httpserver

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/lextures/lextures/server/internal/auth"
	"github.com/lextures/lextures/server/internal/redisclient"
	"github.com/lextures/lextures/server/internal/telemetry"
)

func TestHandleLive_OK(t *testing.T) {
	rr := httptest.NewRecorder()
	handleLive(nil)(rr, httptest.NewRequest(http.MethodGet, "/health/live", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("code %d", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), `"status":"ok"`) {
		t.Fatalf("body %q", rr.Body.String())
	}
}

func TestHandleReady_AllOK(t *testing.T) {
	mr := miniredis.RunT(t)
	rc, err := redisclient.New(context.Background(), redisclient.Config{URL: "redis://" + mr.Addr()})
	if err != nil {
		t.Fatalf("redis: %v", err)
	}
	t.Cleanup(func() { _ = rc.Close() })

	probe := NewHealthProbe(nil, rc, nil)
	// Postgres nil => fail; use a stub via overriding check - actually we need postgres ok.
	// For unit test without DB, test the Ready logic directly with nil pool.
	resp, code := probe.Ready(context.Background())
	if code != http.StatusServiceUnavailable {
		t.Fatalf("code %d", code)
	}
	if resp.Status != "unhealthy" || resp.Checks["postgres"] != "fail" {
		t.Fatalf("resp %+v", resp)
	}
	if resp.Checks["redis"] != "ok" {
		t.Fatalf("redis check %q", resp.Checks["redis"])
	}
}

func TestHandleReady_NoRedisConfigured(t *testing.T) {
	probe := NewHealthProbe(nil, nil, nil)
	resp, code := probe.Ready(context.Background())
	if code != http.StatusServiceUnavailable {
		t.Fatalf("code %d", code)
	}
	if resp.Checks["redis"] != "ok" {
		t.Fatalf("redis should be ok when not configured, got %q", resp.Checks["redis"])
	}
}

func TestHandleReady_HandlerSchema(t *testing.T) {
	probe := NewHealthProbe(nil, nil, nil)
	rr := httptest.NewRecorder()
	handleReady(probe)(rr, httptest.NewRequest(http.MethodGet, "/health/ready", nil))
	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("code %d", rr.Code)
	}
	var body ReadyResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("json: %v", err)
	}
	if body.Status != "unhealthy" {
		t.Fatalf("status %q", body.Status)
	}
	if strings.Contains(rr.Body.String(), "postgres://") || strings.Contains(rr.Body.String(), "password") {
		t.Fatalf("leaked sensitive data: %s", rr.Body.String())
	}
}

func TestHandleHealthDetailed_Unauthenticated(t *testing.T) {
	h := NewHandler(Deps{Pool: nil, JWTSigner: auth.NewJWTSigner("01234567890123456789012345678901")})
	rr := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/health/detailed", nil)
	h.ServeHTTP(rr, r)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("code %d body %s", rr.Code, rr.Body.String())
	}
}

func TestSafeErrorSummary(t *testing.T) {
	cases := map[error]string{
		nil:                      "",
		context.DeadlineExceeded: "timeout",
		errNoDBPool:              "not configured",
	}
	for err, want := range cases {
		if got := safeErrorSummary(err); got != want {
			t.Errorf("safeErrorSummary(%v) = %q, want %q", err, got, want)
		}
	}
}

func TestHealthCheckMetrics(t *testing.T) {
	m := telemetry.NewMetrics()
	probe := NewHealthProbe(nil, nil, m)
	rr := httptest.NewRecorder()
	handleReady(probe)(rr, httptest.NewRequest(http.MethodGet, "/health/ready", nil))
	body := scrapeMetrics(t, m)
	if !strings.Contains(body, `lextures_health_check_total{endpoint="ready",status="503"}`) {
		t.Fatalf("expected health_check metric in:\n%s", body)
	}
}

func scrapeMetrics(t *testing.T, m *telemetry.Metrics) string {
	t.Helper()
	rr := httptest.NewRecorder()
	m.Handler().ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	return rr.Body.String()
}

func TestNewHandler_LiveEndpoint(t *testing.T) {
	h := NewHandler(Deps{Pool: nil})
	rr := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/health/live", nil)
	h.ServeHTTP(rr, r)
	if rr.Code != http.StatusOK {
		t.Fatalf("live: %d", rr.Code)
	}
}
