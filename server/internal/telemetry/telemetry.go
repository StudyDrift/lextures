package telemetry

import (
	"context"
	"log/slog"
	"net/http"
	"time"
)

// Config is the full observability configuration assembled from app config.
type Config struct {
	ServiceName string
	Version     string
	Environment string

	OTel   OTelConfig
	Sentry SentryConfig
}

// Telemetry is the initialised observability stack: a Prometheus registry, an
// OTel tracer provider, and a Sentry client. All pieces are optional and fail
// open — a nil or disabled component never blocks request handling
// (plan 17.7 NFR Reliability).
type Telemetry struct {
	Metrics *Metrics

	sentryEnabled bool
	sentryFlush   func(time.Duration)
	traceShutdown func(context.Context) error
}

// Init builds the telemetry stack: metrics registry, distributed tracing, and
// Sentry. It never returns a fatal error for a misconfigured exporter — those
// are logged and the corresponding component is disabled — so a broken
// observability backend cannot prevent the server from starting.
func Init(ctx context.Context, cfg Config) *Telemetry {
	t := &Telemetry{Metrics: NewMetrics()}
	t.Metrics.SetBuildInfo(cfg.Version, cfg.Environment)
	setDefault(t.Metrics)

	// Distributed tracing (FR-2). Disabled when no endpoint configured.
	if cfg.OTel.ServiceName == "" {
		cfg.OTel.ServiceName = cfg.ServiceName
	}
	if cfg.OTel.Environment == "" {
		cfg.OTel.Environment = cfg.Environment
	}
	if cfg.OTel.Version == "" {
		cfg.OTel.Version = cfg.Version
	}
	shutdown, err := setupTracing(ctx, cfg.OTel)
	if err != nil {
		slog.Warn("telemetry: tracing disabled", "err", err)
		t.traceShutdown = func(context.Context) error { return nil }
	} else {
		t.traceShutdown = shutdown
		if cfg.OTel.Endpoint != "" {
			slog.Info("telemetry: tracing enabled", "endpoint", cfg.OTel.Endpoint, "sample_ratio", cfg.OTel.SampleRatio)
		}
	}

	// Error reporting (FR-3/FR-4). Disabled when no DSN.
	if cfg.Sentry.Environment == "" {
		cfg.Sentry.Environment = cfg.Environment
	}
	if cfg.Sentry.Release == "" {
		cfg.Sentry.Release = cfg.Version
	}
	flush, enabled, serr := initSentry(cfg.Sentry)
	if serr != nil {
		slog.Warn("telemetry: sentry disabled", "err", serr)
	}
	t.sentryEnabled = enabled
	t.sentryFlush = flush
	if enabled {
		slog.Info("telemetry: sentry enabled", "environment", cfg.Sentry.Environment)
	}
	return t
}

// RegisterSources wires the live DB/Redis/job-queue snapshot closures into a
// collector registered on the metrics registry (plan 17.7 FR-1).
func (t *Telemetry) RegisterSources(s Sources) error {
	return t.Metrics.RegisterCollector(newResourceCollector(s))
}

// MetricsHandler is the GET /metrics exposition handler for the internal port.
func (t *Telemetry) MetricsHandler() http.Handler { return t.Metrics.Handler() }

// WrapSlog chains the Sentry ERROR-forwarding handler around inner when Sentry
// is enabled; otherwise it returns inner unchanged. Plumbed through
// logging.Configure so the bridge sits downstream of PII redaction.
func (t *Telemetry) WrapSlog(inner slog.Handler) slog.Handler {
	if !t.sentryEnabled {
		return inner
	}
	return newSentrySlogHandler(inner)
}

// ObserveMiddlewares returns the observation chain (outermost first) to install
// at the top of the router: OTel span → trace-id header → metrics. The span must
// exist before the trace-id header is written, and metrics should observe the
// final status. The Sentry panic-recover middleware is installed separately
// (SentryRecoverMiddleware) so it can sit inside chi's Recoverer and capture the
// panic before Recoverer converts it to a 500.
func (t *Telemetry) ObserveMiddlewares() []func(http.Handler) http.Handler {
	return []func(http.Handler) http.Handler{
		OTelHTTPMiddleware,
		TraceIDMiddleware,
		t.Metrics.MetricsMiddleware,
	}
}

// Shutdown flushes traces and Sentry events with a bounded timeout. Called on
// graceful shutdown; safe to call when components are disabled.
func (t *Telemetry) Shutdown(ctx context.Context) {
	if t.traceShutdown != nil {
		_ = t.traceShutdown(ctx)
	}
	if t.sentryFlush != nil {
		t.sentryFlush(2 * time.Second)
	}
}
