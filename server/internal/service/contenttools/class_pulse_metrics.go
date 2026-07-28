package contenttools

import (
	"strconv"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	classPulseMetricsOnce sync.Once

	classPulseVotesTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "lextures",
		Name:      "content_tool_votes_total",
		Help:      "Class Pulse vote outcomes by round and outcome (CT.21).",
	}, []string{"round", "outcome"})

	classPulseSuppressionHitsTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: "lextures",
		Name:      "content_tool_class_pulse_suppression_hits_total",
		Help:      "Class Pulse aggregate responses withheld for small-n (CT.21).",
	})

	classPulseAggregateCacheHitsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "lextures",
		Name:      "content_tool_class_pulse_aggregate_cache_total",
		Help:      "Class Pulse aggregate cache hits/misses (CT.21).",
	}, []string{"result"})

	classPulseRevoteShiftMagnitude = prometheus.NewHistogram(prometheus.HistogramOpts{
		Namespace: "lextures",
		Name:      "content_tool_class_pulse_revote_shift_magnitude",
		Help:      "Fraction of round-2 voters who changed option (CT.21).",
		Buckets:   []float64{0, 0.1, 0.25, 0.5, 0.75, 1},
	})
)

func registerClassPulseMetrics() {
	classPulseMetricsOnce.Do(func() {
		prometheus.MustRegister(
			classPulseVotesTotal,
			classPulseSuppressionHitsTotal,
			classPulseAggregateCacheHitsTotal,
			classPulseRevoteShiftMagnitude,
		)
		classPulseVotesTotal.WithLabelValues("1", "_reserved").Add(0)
		classPulseAggregateCacheHitsTotal.WithLabelValues("_reserved").Add(0)
	})
}

// ObserveClassPulseVote increments lextures_content_tool_votes_total{round,outcome}.
func ObserveClassPulseVote(round int, outcome string) {
	registerClassPulseMetrics()
	if outcome == "" {
		outcome = "_unknown"
	}
	classPulseVotesTotal.WithLabelValues(strconv.Itoa(round), outcome).Inc()
}

// ObserveClassPulseSuppressionHit increments small-n suppression counter.
func ObserveClassPulseSuppressionHit() {
	registerClassPulseMetrics()
	classPulseSuppressionHitsTotal.Inc()
}

// ObserveClassPulseAggregateCache records cache hit/miss.
func ObserveClassPulseAggregateCache(hit bool) {
	registerClassPulseMetrics()
	if hit {
		classPulseAggregateCacheHitsTotal.WithLabelValues("hit").Inc()
	} else {
		classPulseAggregateCacheHitsTotal.WithLabelValues("miss").Inc()
	}
}

// ObserveClassPulseRevoteShift records the fraction of revoters who changed option.
func ObserveClassPulseRevoteShift(magnitude float64) {
	registerClassPulseMetrics()
	if magnitude < 0 {
		magnitude = 0
	}
	if magnitude > 1 {
		magnitude = 1
	}
	classPulseRevoteShiftMagnitude.Observe(magnitude)
}
