package analytics

import (
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	metricsOnce sync.Once

	summaryWrites = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "lextures",
		Name:      "content_tool_summary_writes_total",
		Help:      "Content tool summary projection writes by tool_id and outcome.",
	}, []string{"tool_id", "outcome"})

	aggregateQuerySeconds = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "lextures",
		Name:      "content_tool_aggregate_query_seconds",
		Help:      "Content tool aggregate query latency by scope.",
		Buckets:   []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.15, 0.25, 0.5, 1},
	}, []string{"scope"})

	gradebookPushes = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "lextures",
		Name:      "content_tool_gradebook_pushes_total",
		Help:      "Content tool gradebook passback outcomes by tool_id and outcome.",
	}, []string{"tool_id", "outcome"})

	xapiStatements = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "lextures",
		Name:      "content_tool_xapi_statements_total",
		Help:      "Content tool xAPI statements emitted by verb.",
	}, []string{"verb"})
)

func ensureMetrics() {
	metricsOnce.Do(func() {
		prometheus.MustRegister(summaryWrites, aggregateQuerySeconds, gradebookPushes, xapiStatements)
	})
}

// IncSummaryWrite increments lextures_content_tool_summary_writes_total.
func IncSummaryWrite(toolID, outcome string) {
	ensureMetrics()
	if toolID == "" {
		toolID = "unknown"
	}
	if outcome == "" {
		outcome = "unknown"
	}
	summaryWrites.WithLabelValues(toolID, outcome).Inc()
}

// ObserveAggregate records aggregate query latency.
func ObserveAggregate(scope string, d time.Duration) {
	ensureMetrics()
	if scope == "" {
		scope = "unknown"
	}
	aggregateQuerySeconds.WithLabelValues(scope).Observe(d.Seconds())
}

// IncGradebookPush increments lextures_content_tool_gradebook_pushes_total.
func IncGradebookPush(toolID, outcome string) {
	ensureMetrics()
	if toolID == "" {
		toolID = "unknown"
	}
	if outcome == "" {
		outcome = "unknown"
	}
	gradebookPushes.WithLabelValues(toolID, outcome).Inc()
}

// IncXAPI increments lextures_content_tool_xapi_statements_total.
func IncXAPI(verb string) {
	ensureMetrics()
	if verb == "" {
		verb = "unknown"
	}
	xapiStatements.WithLabelValues(verb).Inc()
}
