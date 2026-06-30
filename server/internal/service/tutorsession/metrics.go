package tutorsession

import (
	"sync/atomic"
	"time"
)

// Metrics tracks tutor observability counters (plan 19.1 NFR observability).
type Metrics struct {
	requestsTotal atomic.Uint64
	latencyTotal  atomic.Uint64
	latencyCount  atomic.Uint64
	citationTotal atomic.Uint64
	citationCount atomic.Uint64
}

var globalMetrics Metrics

// RecordRequest increments tutor_requests_total.
func RecordRequest() {
	globalMetrics.requestsTotal.Add(1)
}

// RecordLatency adds a latency sample in milliseconds.
func RecordLatency(ms int64) {
	if ms < 0 {
		return
	}
	globalMetrics.latencyTotal.Add(uint64(ms))
	globalMetrics.latencyCount.Add(1)
}

// RecordCitations records citations per response.
func RecordCitations(count int) {
	if count < 0 {
		return
	}
	globalMetrics.citationTotal.Add(uint64(count))
	globalMetrics.citationCount.Add(1)
}

// Snapshot returns current metric values for diagnostics.
func Snapshot() map[string]uint64 {
	out := map[string]uint64{
		"tutor_requests_total": globalMetrics.requestsTotal.Load(),
	}
	if n := globalMetrics.latencyCount.Load(); n > 0 {
		out["tutor_latency_ms_avg"] = globalMetrics.latencyTotal.Load() / n
	}
	if n := globalMetrics.citationCount.Load(); n > 0 {
		out["tutor_citations_per_response_avg"] = globalMetrics.citationTotal.Load() / n
	}
	return out
}

// SinceStart returns elapsed ms since start (for latency measurement).
func SinceStart(start time.Time) int64 {
	return time.Since(start).Milliseconds()
}
