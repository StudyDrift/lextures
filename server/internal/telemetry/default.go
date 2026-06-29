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

// RecordAIProvider records an AI provider call on the default instance
// (plan 16.7 / 17.7 §11). No-op when telemetry is not initialised.
func RecordAIProvider(provider, model, outcome string, seconds, costDollars float64) {
	if m := defaultMetrics.Load(); m != nil {
		m.ObserveAIProvider(provider, model, outcome, seconds, costDollars)
	}
}
