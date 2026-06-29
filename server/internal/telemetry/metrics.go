// Package telemetry is the observability layer (plan 17.7): Prometheus metrics,
// OpenTelemetry traces, and Sentry error reporting. It is intentionally
// decoupled from the rest of the server — callers pass live resource snapshots
// via Sources closures rather than telemetry importing db/redis/jobqueue — so a
// failure in the observability pipeline can never affect request handling
// (plan 17.7 NFR Reliability).
package telemetry

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// namespace prefixes every Lextures-authored metric so operators can filter
// application metrics from runtime/process metrics in Grafana.
const namespace = "lextures"

// durationBuckets are latency histogram buckets tuned for an HTTP API: sub-10ms
// hot paths up to multi-second slow endpoints. Kept deliberately small to bound
// Prometheus storage (plan 17.7 risk: high-cardinality series).
var durationBuckets = []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10}

// Metrics holds the application metric vectors and the registry that exposes
// them. All counters/histograms are registered once at startup; adding a new
// metric is one field plus one registration here (plan 17.7 NFR Maintainability).
type Metrics struct {
	registry *prometheus.Registry

	httpRequests        *prometheus.CounterVec
	httpDuration        *prometheus.HistogramVec
	httpInFlight        prometheus.Gauge
	aiProviderCalls     *prometheus.CounterVec
	aiProviderLatency   *prometheus.HistogramVec
	aiProviderCostTotal *prometheus.CounterVec
	businessEvents      *prometheus.CounterVec
	healthChecks        *prometheus.CounterVec
	buildInfo           *prometheus.GaugeVec
}

// NewMetrics builds a self-contained registry (not the global default, so tests
// can construct independent instances) with the application metrics and the
// standard Go runtime + process collectors registered.
func NewMetrics() *Metrics {
	reg := prometheus.NewRegistry()
	m := &Metrics{
		registry: reg,
		httpRequests: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "http_requests_total",
			Help:      "Total HTTP requests by method, route group, and status class.",
		}, []string{"method", "route", "status"}),
		httpDuration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: namespace,
			Name:      "http_request_duration_seconds",
			Help:      "HTTP request latency in seconds by method and route group.",
			Buckets:   durationBuckets,
		}, []string{"method", "route"}),
		httpInFlight: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "http_requests_in_flight",
			Help:      "In-flight HTTP requests currently being served.",
		}),
		aiProviderCalls: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "ai_provider_calls_total",
			Help:      "AI provider calls by provider, model, and outcome (plan 16.7 / 17.7).",
		}, []string{"provider", "model", "outcome"}),
		aiProviderLatency: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: namespace,
			Name:      "ai_provider_latency_seconds",
			Help:      "AI provider call latency in seconds by provider and model.",
			Buckets:   []float64{0.1, 0.25, 0.5, 1, 2.5, 5, 10, 30, 60},
		}, []string{"provider", "model"}),
		aiProviderCostTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "ai_estimated_cost_dollars_total",
			Help:      "Estimated AI spend in US dollars by provider and model.",
		}, []string{"provider", "model"}),
		businessEvents: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "business_events_total",
			Help:      "Business events (enrollments, grade submissions, etc.) by type.",
		}, []string{"event"}),
		healthChecks: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "health_check_total",
			Help:      "Health probe invocations by endpoint and HTTP-style status (plan 17.8).",
		}, []string{"endpoint", "status"}),
		buildInfo: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "build_info",
			Help:      "Build metadata; always 1, labels carry version and environment.",
		}, []string{"version", "env"}),
	}

	reg.MustRegister(
		m.httpRequests,
		m.httpDuration,
		m.httpInFlight,
		m.aiProviderCalls,
		m.aiProviderLatency,
		m.aiProviderCostTotal,
		m.businessEvents,
		m.healthChecks,
		m.buildInfo,
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
	)
	return m
}

// SetBuildInfo records the running version and environment as a build_info gauge.
func (m *Metrics) SetBuildInfo(version, env string) {
	if version == "" {
		version = "dev"
	}
	m.buildInfo.WithLabelValues(version, env).Set(1)
}

// Registry returns the underlying Prometheus registry (used by RegisterCollector
// and the /metrics handler).
func (m *Metrics) Registry() *prometheus.Registry { return m.registry }

// RegisterCollector adds a custom collector (e.g. the live resource collector)
// to the registry. Safe to call once at startup.
func (m *Metrics) RegisterCollector(c prometheus.Collector) error {
	return m.registry.Register(c)
}

// Handler returns the Prometheus exposition handler for GET /metrics. It is
// served on the internal metrics port, never the public LB port (plan 17.7
// FR-1 / NFR Security).
func (m *Metrics) Handler() http.Handler {
	return promhttp.HandlerFor(m.registry, promhttp.HandlerOpts{
		EnableOpenMetrics: false,
		Registry:          m.registry,
	})
}

// ObserveHTTP records one completed HTTP request (plan 17.7 FR-1, AC-1).
func (m *Metrics) ObserveHTTP(method, route, statusClass string, seconds float64) {
	m.httpRequests.WithLabelValues(method, route, statusClass).Inc()
	m.httpDuration.WithLabelValues(method, route).Observe(seconds)
}

// IncInFlight / DecInFlight track concurrent in-flight requests.
func (m *Metrics) IncInFlight() { m.httpInFlight.Inc() }
func (m *Metrics) DecInFlight() { m.httpInFlight.Dec() }

// ObserveAIProvider records an AI provider call's outcome, latency, and
// estimated cost (plan 16.7 / 17.7 §11). outcome is typically "ok" or "error".
func (m *Metrics) ObserveAIProvider(provider, model, outcome string, seconds, costDollars float64) {
	if provider == "" {
		provider = "unknown"
	}
	if model == "" {
		model = "unknown"
	}
	m.aiProviderCalls.WithLabelValues(provider, model, outcome).Inc()
	m.aiProviderLatency.WithLabelValues(provider, model).Observe(seconds)
	if costDollars > 0 {
		m.aiProviderCostTotal.WithLabelValues(provider, model).Add(costDollars)
	}
}

// IncBusinessEvent increments a named business metric (e.g. "enrollment_created",
// "grade_submitted") — plan 17.7 FR-5(e) Business Metrics dashboard.
func (m *Metrics) IncBusinessEvent(event string) {
	if event == "" {
		return
	}
	m.businessEvents.WithLabelValues(event).Inc()
}

// IncHealthCheck records one health probe result (plan 17.8 Observability).
func (m *Metrics) IncHealthCheck(endpoint, status string) {
	if endpoint == "" {
		endpoint = "unknown"
	}
	if status == "" {
		status = "unknown"
	}
	m.healthChecks.WithLabelValues(endpoint, status).Inc()
}
