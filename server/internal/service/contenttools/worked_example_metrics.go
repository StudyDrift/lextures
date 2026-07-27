package contenttools

import (
	"sync"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	workedExampleMetricsOnce sync.Once

	workedExampleStepChecksTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "lextures",
		Name:      "content_tool_step_checks_total",
		Help:      "Worked Example step check outcomes by result (CT.18).",
	}, []string{"result"})

	workedExampleHintsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "lextures",
		Name:      "content_tool_step_hints_total",
		Help:      "Worked Example hint requests by outcome (CT.18).",
	}, []string{"outcome"})

	workedExampleRevealsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "lextures",
		Name:      "content_tool_step_reveals_total",
		Help:      "Worked Example step/all reveals (CT.18).",
	}, []string{"scope"})

	workedExampleUndecidableTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: "lextures",
		Name:      "content_tool_step_normaliser_undecidable_total",
		Help:      "Worked Example expression normaliser undecidable outcomes (CT.18 authoring-quality signal).",
	})
)

func registerWorkedExampleMetrics() {
	workedExampleMetricsOnce.Do(func() {
		prometheus.MustRegister(
			workedExampleStepChecksTotal,
			workedExampleHintsTotal,
			workedExampleRevealsTotal,
			workedExampleUndecidableTotal,
		)
		workedExampleStepChecksTotal.WithLabelValues("_reserved").Add(0)
		workedExampleHintsTotal.WithLabelValues("_reserved").Add(0)
		workedExampleRevealsTotal.WithLabelValues("_reserved").Add(0)
	})
}

// ObserveWorkedExampleCheck increments lextures_content_tool_step_checks_total{result}.
func ObserveWorkedExampleCheck(result string) {
	registerWorkedExampleMetrics()
	if result == "" {
		result = "_unknown"
	}
	workedExampleStepChecksTotal.WithLabelValues(result).Inc()
}

// ObserveWorkedExampleHint increments hint usage counter.
func ObserveWorkedExampleHint(outcome string) {
	registerWorkedExampleMetrics()
	if outcome == "" {
		outcome = "_unknown"
	}
	workedExampleHintsTotal.WithLabelValues(outcome).Inc()
}

// ObserveWorkedExampleReveal increments reveal counter (step|all).
func ObserveWorkedExampleReveal(scope string) {
	registerWorkedExampleMetrics()
	if scope == "" {
		scope = "_unknown"
	}
	workedExampleRevealsTotal.WithLabelValues(scope).Inc()
}

// ObserveWorkedExampleUndecidable increments normaliser-undecidable counter.
func ObserveWorkedExampleUndecidable() {
	registerWorkedExampleMetrics()
	workedExampleUndecidableTotal.Inc()
}
