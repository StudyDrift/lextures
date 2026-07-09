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

	httpRequests         *prometheus.CounterVec
	httpDuration         *prometheus.HistogramVec
	httpInFlight         prometheus.Gauge
	aiProviderCalls      *prometheus.CounterVec
	aiProviderLatency    *prometheus.HistogramVec
	aiProviderCostTotal  *prometheus.CounterVec
	businessEvents       *prometheus.CounterVec
	bannerActive         *prometheus.GaugeVec
	healthChecks         *prometheus.CounterVec
	buildInfo            *prometheus.GaugeVec
	marketplaceFlagState       *prometheus.GaugeVec
	marketplaceListingSaved    *prometheus.CounterVec
	marketplaceStorefrontViews *prometheus.CounterVec
	marketplaceDetailViews     *prometheus.CounterVec
	marketplaceFacetUsage      *prometheus.CounterVec
	marketplaceClaimTotal      *prometheus.CounterVec
	marketplaceCheckoutCreated *prometheus.CounterVec
	marketplacePurchaseCompleted *prometheus.CounterVec
	marketplaceRefundTotal     *prometheus.CounterVec
}

// NewMetrics builds a self-contained registry (not the global default, so tests
// can construct independent instances) with the application metrics and the
// standard Go runtime + process collectors registered.
// deployColor labels HTTP metrics for blue/green canary analysis (plan 17.9).
func NewMetrics(deployColor ...string) *Metrics {
	color := "stable"
	if len(deployColor) > 0 && deployColor[0] != "" {
		color = deployColor[0]
	}
	constLabels := prometheus.Labels{"deploy_color": color}
	reg := prometheus.NewRegistry()
	m := &Metrics{
		registry: reg,
		httpRequests: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace:   namespace,
			Name:        "http_requests_total",
			Help:        "Total HTTP requests by method, route group, and status class.",
			ConstLabels: constLabels,
		}, []string{"method", "route", "status"}),
		httpDuration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Namespace:   namespace,
			Name:        "http_request_duration_seconds",
			Help:        "HTTP request latency in seconds by method and route group.",
			Buckets:     durationBuckets,
			ConstLabels: constLabels,
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
		bannerActive: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "banner_active",
			Help:      "Whether a maintenance banner is currently active (1) or not (0) by scope and severity (plan 18.6).",
		}, []string{"scope", "severity"}),
		healthChecks: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "health_check_total",
			Help:      "Health probe invocations by endpoint and HTTP-style status (plan 17.8).",
		}, []string{"endpoint", "status"}),
		buildInfo: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace:   namespace,
			Name:        "build_info",
			Help:        "Build metadata; always 1, labels carry version and environment.",
			ConstLabels: constLabels,
		}, []string{"version", "env"}),
		marketplaceFlagState: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "marketplace_flag_state",
			Help:      "Whether the in-app course marketplace flag is enabled (1) or disabled (0) (plan MKT1).",
		}, []string{"enabled"}),
		marketplaceListingSaved: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "marketplace_listing_saved_total",
			Help:      "Course marketplace listing saves by listed and free state (plan MKT2).",
		}, []string{"listed", "free"}),
		marketplaceStorefrontViews: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "marketplace_storefront_view_total",
			Help:      "Authenticated marketplace storefront list views (plan MKT3).",
		}, []string{}),
		marketplaceDetailViews: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "marketplace_detail_view_total",
			Help:      "Marketplace course detail views by ownership (plan MKT3).",
		}, []string{"owned"}),
		marketplaceFacetUsage: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "marketplace_facet_usage_total",
			Help:      "Marketplace storefront searches that used a filter facet (plan MKT3).",
		}, []string{}),
		marketplaceClaimTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "marketplace_claim_total",
			Help:      "Marketplace free-claim attempts by result (plan MKT4).",
		}, []string{"result"}),
		marketplaceCheckoutCreated: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "marketplace_checkout_created",
			Help:      "Marketplace paid checkout sessions created (plan MKT4).",
		}, []string{}),
		marketplacePurchaseCompleted: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "marketplace_purchase_completed",
			Help:      "Marketplace paid purchases completed via webhook (plan MKT4).",
		}, []string{}),
		marketplaceRefundTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "marketplace_refund_total",
			Help:      "Marketplace course purchase refunds processed (plan MKT4).",
		}, []string{}),
	}

	reg.MustRegister(
		m.httpRequests,
		m.httpDuration,
		m.httpInFlight,
		m.aiProviderCalls,
		m.aiProviderLatency,
		m.aiProviderCostTotal,
		m.businessEvents,
		m.bannerActive,
		m.healthChecks,
		m.buildInfo,
		m.marketplaceFlagState,
		m.marketplaceListingSaved,
		m.marketplaceStorefrontViews,
		m.marketplaceDetailViews,
		m.marketplaceFacetUsage,
		m.marketplaceClaimTotal,
		m.marketplaceCheckoutCreated,
		m.marketplacePurchaseCompleted,
		m.marketplaceRefundTotal,
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

// SetBannerActive records the currently served maintenance banner gauge (plan 18.6).
// Pass empty scope to clear all known label combinations to zero.
func (m *Metrics) SetBannerActive(scope, severity string) {
	for _, sc := range []string{"global", "org"} {
		for _, sev := range []string{"info", "warning", "error"} {
			m.bannerActive.WithLabelValues(sc, sev).Set(0)
		}
	}
	if scope == "" {
		return
	}
	if severity == "" {
		severity = "info"
	}
	m.bannerActive.WithLabelValues(scope, severity).Set(1)
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

// SetMarketplaceFlagState records the course marketplace platform flag (plan MKT1).
// Emits marketplace_flag_state{enabled="true"|"false"} with value 1 for the active state.
func (m *Metrics) SetMarketplaceFlagState(enabled bool) {
	if m == nil || m.marketplaceFlagState == nil {
		return
	}
	m.marketplaceFlagState.WithLabelValues("true").Set(0)
	m.marketplaceFlagState.WithLabelValues("false").Set(0)
	if enabled {
		m.marketplaceFlagState.WithLabelValues("true").Set(1)
	} else {
		m.marketplaceFlagState.WithLabelValues("false").Set(1)
	}
}

// RecordMarketplaceListingSaved increments marketplace_listing_saved_total (plan MKT2).
func (m *Metrics) RecordMarketplaceListingSaved(listed, free bool) {
	if m == nil || m.marketplaceListingSaved == nil {
		return
	}
	m.marketplaceListingSaved.WithLabelValues(boolLabel(listed), boolLabel(free)).Inc()
}

// RecordMarketplaceStorefrontView increments marketplace_storefront_view_total (plan MKT3).
func (m *Metrics) RecordMarketplaceStorefrontView() {
	if m == nil || m.marketplaceStorefrontViews == nil {
		return
	}
	m.marketplaceStorefrontViews.WithLabelValues().Inc()
}

// RecordMarketplaceDetailView increments marketplace_detail_view_total{owned} (plan MKT3).
func (m *Metrics) RecordMarketplaceDetailView(owned bool) {
	if m == nil || m.marketplaceDetailViews == nil {
		return
	}
	m.marketplaceDetailViews.WithLabelValues(boolLabel(owned)).Inc()
}

// RecordMarketplaceFacetUsage increments marketplace_facet_usage_total (plan MKT3).
func (m *Metrics) RecordMarketplaceFacetUsage() {
	if m == nil || m.marketplaceFacetUsage == nil {
		return
	}
	m.marketplaceFacetUsage.WithLabelValues().Inc()
}

// RecordMarketplaceClaim increments marketplace_claim_total{result} (plan MKT4).
func (m *Metrics) RecordMarketplaceClaim(result string) {
	if m == nil || m.marketplaceClaimTotal == nil {
		return
	}
	if result == "" {
		result = "unknown"
	}
	m.marketplaceClaimTotal.WithLabelValues(result).Inc()
}

// RecordMarketplaceCheckoutCreated increments marketplace_checkout_created (plan MKT4).
func (m *Metrics) RecordMarketplaceCheckoutCreated() {
	if m == nil || m.marketplaceCheckoutCreated == nil {
		return
	}
	m.marketplaceCheckoutCreated.WithLabelValues().Inc()
}

// RecordMarketplacePurchaseCompleted increments marketplace_purchase_completed (plan MKT4).
func (m *Metrics) RecordMarketplacePurchaseCompleted() {
	if m == nil || m.marketplacePurchaseCompleted == nil {
		return
	}
	m.marketplacePurchaseCompleted.WithLabelValues().Inc()
}

// RecordMarketplaceRefund increments marketplace_refund_total (plan MKT4).
func (m *Metrics) RecordMarketplaceRefund() {
	if m == nil || m.marketplaceRefundTotal == nil {
		return
	}
	m.marketplaceRefundTotal.WithLabelValues().Inc()
}

func boolLabel(v bool) string {
	if v {
		return "true"
	}
	return "false"
}
