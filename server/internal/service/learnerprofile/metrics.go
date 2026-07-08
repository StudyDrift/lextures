package learnerprofile

import (
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	metricsOnce sync.Once
	recomputeTotal *prometheus.CounterVec
	recomputeDuration *prometheus.HistogramVec
	facetsPopulated prometheus.Gauge
	controlTotal *prometheus.CounterVec
	adaptationTotal *prometheus.CounterVec
)

func initMetrics() {
	recomputeTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "lextures",
		Name:      "learner_profile_recompute_total",
		Help:      "Learner profile facet recomputes by facet, mode, and result.",
	}, []string{"facet", "mode", "result"})
	recomputeDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "lextures",
		Name:      "learner_profile_recompute_duration_seconds",
		Help:      "Learner profile facet recompute duration in seconds.",
		Buckets:   []float64{0.01, 0.05, 0.1, 0.25, 0.5, 1, 2, 5},
	}, []string{"facet"})
	facetsPopulated = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "lextures",
		Name:      "learner_profile_facets_populated",
		Help:      "Number of learner profile facets in ok state across the fleet.",
	})
	controlTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "lextures",
		Name:      "learner_profile_control_total",
		Help:      "Learner profile privacy control actions by action.",
	}, []string{"action"})
	adaptationTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "lextures",
		Name:      "learner_profile_adaptation_total",
		Help:      "Profile-powered adaptivity applications by consumer and result.",
	}, []string{"consumer", "result"})
}

// RegisterMetrics registers learner profile metrics with reg (no-op when reg is nil).
func RegisterMetrics(reg prometheus.Registerer) {
	if reg == nil {
		return
	}
	metricsOnce.Do(func() {
		initMetrics()
		reg.MustRegister(recomputeTotal, recomputeDuration, facetsPopulated, controlTotal, adaptationTotal)
	})
}

// RecordControl increments the learner_profile_control_total metric.
func RecordControl(action string) {
	if controlTotal != nil {
		controlTotal.WithLabelValues(action).Inc()
	}
}

func recordRecompute(facet, mode, result string, started time.Time) {
	if recomputeTotal == nil {
		return
	}
	recomputeTotal.WithLabelValues(facet, mode, result).Inc()
	recomputeDuration.WithLabelValues(facet).Observe(time.Since(started).Seconds())
}

func setFacetsPopulated(n float64) {
	if facetsPopulated != nil {
		facetsPopulated.Set(n)
	}
}

func recordAdaptation(consumer, result string) {
	if adaptationTotal != nil {
		adaptationTotal.WithLabelValues(consumer, result).Inc()
	}
}

// RecordAdaptation increments learner_profile_adaptation_total (for httpserver wiring).
func RecordAdaptation(consumer, result string) {
	recordAdaptation(consumer, result)
}