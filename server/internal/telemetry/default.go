package telemetry

import "sync/atomic"

// defaultMetrics holds the process-wide Metrics instance set by Init, so service
// code can emit business and AI metrics via the package-level helpers below
// without threading a *Metrics through every call site. It is nil until Init
// runs (and in unit tests that never call Init), in which case the helpers are
// safe no-ops.
var defaultMetrics atomic.Pointer[Metrics]

// setDefault installs m as the process-wide metrics instance (called by Init).
func setDefault(m *Metrics) { defaultMetrics.Store(m) }

// Default returns the process-wide Metrics, or nil if telemetry is not started.
func Default() *Metrics { return defaultMetrics.Load() }

// RecordBusinessEvent increments a business metric on the default instance
// (plan 17.7 FR-5e). No-op when telemetry is not initialised.
func RecordBusinessEvent(event string) {
	if m := defaultMetrics.Load(); m != nil {
		m.IncBusinessEvent(event)
	}
}

// SetMarketplaceFlagState records the course marketplace flag on the default
// metrics instance (plan MKT1). No-op when telemetry is not initialised.
func SetMarketplaceFlagState(enabled bool) {
	if m := defaultMetrics.Load(); m != nil {
		m.SetMarketplaceFlagState(enabled)
	}
}

// RecordMarketplaceListingSaved records a marketplace listing save on the default
// metrics instance (plan MKT2). No-op when telemetry is not initialised.
func RecordMarketplaceListingSaved(listed, free bool) {
	if m := defaultMetrics.Load(); m != nil {
		m.RecordMarketplaceListingSaved(listed, free)
	}
}

// RecordMarketplaceStorefrontView records a storefront list view (plan MKT3).
func RecordMarketplaceStorefrontView() {
	if m := defaultMetrics.Load(); m != nil {
		m.RecordMarketplaceStorefrontView()
	}
}

// RecordMarketplaceDetailView records a marketplace detail view (plan MKT3).
func RecordMarketplaceDetailView(owned bool) {
	if m := defaultMetrics.Load(); m != nil {
		m.RecordMarketplaceDetailView(owned)
	}
}

// RecordMarketplaceFacetUsage records a filtered storefront search (plan MKT3).
func RecordMarketplaceFacetUsage() {
	if m := defaultMetrics.Load(); m != nil {
		m.RecordMarketplaceFacetUsage()
	}
}

// RecordMarketplaceClaim records a free-claim attempt result (plan MKT4).
func RecordMarketplaceClaim(result string) {
	if m := defaultMetrics.Load(); m != nil {
		m.RecordMarketplaceClaim(result)
	}
}

// RecordMarketplaceCheckoutCreated records a paid checkout session (plan MKT4).
func RecordMarketplaceCheckoutCreated() {
	if m := defaultMetrics.Load(); m != nil {
		m.RecordMarketplaceCheckoutCreated()
	}
}

// RecordMarketplacePurchaseCompleted records a paid purchase via webhook (plan MKT4).
func RecordMarketplacePurchaseCompleted() {
	if m := defaultMetrics.Load(); m != nil {
		m.RecordMarketplacePurchaseCompleted()
	}
}

// RecordMarketplaceRefund records a marketplace course refund (plan MKT4).
func RecordMarketplaceRefund() {
	if m := defaultMetrics.Load(); m != nil {
		m.RecordMarketplaceRefund()
	}
}

// RecordAIProvider records an AI provider call on the default instance
// (plan 16.7 / 17.7 §11). No-op when telemetry is not initialised.
func RecordAIProvider(provider, model, outcome string, seconds, costDollars float64) {
	if m := defaultMetrics.Load(); m != nil {
		m.ObserveAIProvider(provider, model, outcome, seconds, costDollars)
	}
}
