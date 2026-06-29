package telemetry

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestInit_DisabledStackStillServesMetrics(t *testing.T) {
	tel := Init(context.Background(), Config{ServiceName: "test", Version: "9.9", Environment: "test"})
	if tel.Metrics == nil {
		t.Fatal("metrics must always be initialised")
	}
	if tel.sentryEnabled {
		t.Error("sentry should be disabled without DSN")
	}
	// /metrics handler works with no exporters configured.
	rr := httptest.NewRecorder()
	tel.MetricsHandler().ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("metrics status %d", rr.Code)
	}
	tel.Shutdown(context.Background()) // must not panic when disabled
}

func TestInit_ObserveMiddlewaresCount(t *testing.T) {
	tel := Init(context.Background(), Config{})
	if got := len(tel.ObserveMiddlewares()); got != 3 {
		t.Errorf("ObserveMiddlewares len = %d, want 3", got)
	}
}

func TestWrapSlog_IdentityWhenSentryDisabled(t *testing.T) {
	tel := &Telemetry{sentryEnabled: false}
	inner := slog.NewJSONHandler(httptest.NewRecorder(), nil)
	if got := tel.WrapSlog(inner); got != inner {
		t.Error("WrapSlog should return inner unchanged when Sentry is disabled")
	}
}

func TestWrapSlog_WrapsWhenSentryEnabled(t *testing.T) {
	tel := &Telemetry{sentryEnabled: true}
	inner := slog.NewJSONHandler(httptest.NewRecorder(), nil)
	got := tel.WrapSlog(inner)
	if _, ok := got.(*sentrySlogHandler); !ok {
		t.Errorf("WrapSlog should wrap in sentrySlogHandler, got %T", got)
	}
}

func TestRegisterSources(t *testing.T) {
	tel := Init(context.Background(), Config{})
	if err := tel.RegisterSources(Sources{
		DBPool: func() DBPoolSnapshot { return DBPoolSnapshot{Total: 1, Max: 2} },
	}); err != nil {
		t.Fatalf("register: %v", err)
	}
}
