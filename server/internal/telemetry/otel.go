package telemetry

import (
	"context"
	"net/http"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"
)

// OTelConfig configures distributed tracing (plan 17.7 FR-2). When Endpoint is
// empty, tracing is disabled and a no-op tracer provider is used so spans add
// zero overhead.
type OTelConfig struct {
	// Endpoint is the OTLP/HTTP collector endpoint (host:port). Empty disables tracing.
	Endpoint string
	// Insecure sends plaintext OTLP (in-VPC collector); production may keep TLS.
	Insecure bool
	// ServiceName labels all spans (e.g. "lextures-api").
	ServiceName string
	// Environment is the deployment environment attribute (staging/production).
	Environment string
	// Version is the service version attribute.
	Version string
	// SampleRatio is the head-based sample rate (0..1). 0.1 = 10% in production,
	// 1.0 = 100% in staging (plan 17.7 NFR Performance / open question 2).
	SampleRatio float64
}

// setupTracing installs a global OTel tracer provider exporting to an OTLP/HTTP
// collector with an async batch processor (non-blocking; plan 17.7 NFR
// Reliability). Returns a shutdown func that flushes pending spans. When
// disabled it returns a no-op shutdown.
func setupTracing(ctx context.Context, cfg OTelConfig) (func(context.Context) error, error) {
	// Always install the W3C trace-context propagator so trace IDs flow across
	// services and the X-Trace-Id header is populated even with sampling off.
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{}, propagation.Baggage{},
	))

	if cfg.Endpoint == "" {
		return func(context.Context) error { return nil }, nil
	}

	opts := []otlptracehttp.Option{otlptracehttp.WithEndpoint(cfg.Endpoint)}
	if cfg.Insecure {
		opts = append(opts, otlptracehttp.WithInsecure())
	}
	exporter, err := otlptracehttp.New(ctx, opts...)
	if err != nil {
		return nil, err
	}

	res, err := resource.Merge(resource.Default(), resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.ServiceName(serviceNameOr(cfg.ServiceName)),
		semconv.ServiceVersion(cfg.Version),
		semconv.DeploymentEnvironment(cfg.Environment),
	))
	if err != nil {
		res = resource.Default()
	}

	ratio := cfg.SampleRatio
	if ratio <= 0 {
		ratio = 0.1
	}
	if ratio > 1 {
		ratio = 1
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter, sdktrace.WithBatchTimeout(5*time.Second)),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.ParentBased(sdktrace.TraceIDRatioBased(ratio))),
	)
	otel.SetTracerProvider(tp)
	return tp.Shutdown, nil
}

func serviceNameOr(name string) string {
	if name == "" {
		return "lextures-api"
	}
	return name
}

// OTelHTTPMiddleware wraps handlers so every request is a root/child span with
// the route as the span name (plan 17.7 FR-2, AC-2). It uses the chi route
// pattern for the span name so trace search groups by endpoint, not raw path.
func OTelHTTPMiddleware(next http.Handler) http.Handler {
	instrumented := otelhttp.NewHandler(next, "http.server",
		otelhttp.WithSpanNameFormatter(func(_ string, r *http.Request) string {
			return r.Method + " " + spanRouteName(r)
		}),
	)
	return instrumented
}

func spanRouteName(r *http.Request) string {
	return routeLabel(r)
}

// Tracer returns a named tracer from the global provider for manual spans in
// service code (plan 17.7 FR-2: child spans for DB queries and external calls).
func Tracer(name string) trace.Tracer {
	return otel.Tracer(name)
}

// SpanAttr is a convenience for non-PII span attributes.
func SpanAttr(key, value string) attribute.KeyValue {
	return attribute.String(key, value)
}
