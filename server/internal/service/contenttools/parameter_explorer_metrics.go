package contenttools

import (
	"sync"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	parameterExplorerMetricsOnce sync.Once

	parameterExplorerCheckpointsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "lextures",
		Name:      "content_tool_explorer_checkpoints_total",
		Help:      "Parameter Explorer checkpoint outcomes (CT.16).",
	}, []string{"outcome"})

	parameterExplorerAnswersTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "lextures",
		Name:      "content_tool_explorer_answers_total",
		Help:      "Parameter Explorer answer submit outcomes (CT.16).",
	}, []string{"outcome"})

	parameterExplorerResetsTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: "lextures",
		Name:      "content_tool_explorer_resets_total",
		Help:      "Parameter Explorer in-tool reset_defaults actions (CT.16).",
	})
)

func registerParameterExplorerMetrics() {
	parameterExplorerMetricsOnce.Do(func() {
		prometheus.MustRegister(
			parameterExplorerCheckpointsTotal,
			parameterExplorerAnswersTotal,
			parameterExplorerResetsTotal,
		)
		parameterExplorerCheckpointsTotal.WithLabelValues("_reserved").Add(0)
		parameterExplorerAnswersTotal.WithLabelValues("_reserved").Add(0)
	})
}

// ObserveParameterExplorerCheckpoint increments checkpoint outcomes (CT.16).
func ObserveParameterExplorerCheckpoint(outcome string) {
	registerParameterExplorerMetrics()
	if outcome == "" {
		outcome = "_unknown"
	}
	parameterExplorerCheckpointsTotal.WithLabelValues(outcome).Inc()
}

// ObserveParameterExplorerAnswer increments answer submit outcomes (CT.16).
func ObserveParameterExplorerAnswer(outcome string) {
	registerParameterExplorerMetrics()
	if outcome == "" {
		outcome = "_unknown"
	}
	parameterExplorerAnswersTotal.WithLabelValues(outcome).Inc()
}

// ObserveParameterExplorerReset increments in-tool reset_defaults (CT.16).
func ObserveParameterExplorerReset() {
	registerParameterExplorerMetrics()
	parameterExplorerResetsTotal.Inc()
}
