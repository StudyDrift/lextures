package contenttools

import (
	"sync"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	metricsOnce sync.Once

	instancesTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "lextures",
		Name:      "content_tool_instances_total",
		Help:      "Content tool instance mutations by tool_id and action.",
	}, []string{"tool_id", "action"})

	configValidationFailures = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "lextures",
		Name:      "content_tool_config_validation_failures_total",
		Help:      "Config JSON Schema validation failures by tool_id.",
	}, []string{"tool_id"})

	insertTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "lextures",
		Name:      "content_tool_insert_total",
		Help:      "Content tool inserts by tool_id and surface (toolbar|slash|paste|duplicate|api).",
	}, []string{"tool_id", "surface"})

	configSaveTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "lextures",
		Name:      "content_tool_config_save_total",
		Help:      "Content tool config save outcomes by tool_id and outcome.",
	}, []string{"tool_id", "outcome"})

	registrySize = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "lextures",
		Name:      "content_tool_registry_size",
		Help:      "Number of tools loaded in the Content Tools registry.",
	})

	killSwitchEngaged = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "lextures",
		Name:      "content_tools_kill_switch_engaged",
		Help:      "1 when CONTENT_TOOLS_KILL_SWITCH is engaged, else 0.",
	})

	stateSavesTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "lextures",
		Name:      "content_tool_state_saves_total",
		Help:      "Content tool state save outcomes by tool_id and outcome.",
	}, []string{"tool_id", "outcome"})

	stateConflictsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "lextures",
		Name:      "content_tool_state_conflicts_total",
		Help:      "Content tool state revision conflicts by tool_id.",
	}, []string{"tool_id"})

	actionLatency = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "lextures",
		Name:      "content_tool_action_latency_seconds",
		Help:      "Content tool server action latency by tool_id and action.",
		Buckets:   []float64{0.01, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10},
	}, []string{"tool_id", "action"})

	renderErrorsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "lextures",
		Name:      "content_tool_render_errors_total",
		Help:      "Client-reported content tool render errors by tool_id (reserved; incremented via telemetry ingest).",
	}, []string{"tool_id"})

	offlineReplaysTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "lextures",
		Name:      "content_tool_offline_replays_total",
		Help:      "Offline outbox replay outcomes for content tool writes.",
	}, []string{"outcome"})
)

func registerMetrics() {
	metricsOnce.Do(func() {
		prometheus.MustRegister(
			instancesTotal, configValidationFailures, insertTotal, configSaveTotal,
			registrySize, killSwitchEngaged,
			stateSavesTotal, stateConflictsTotal, actionLatency, renderErrorsTotal, offlineReplaysTotal,
		)
		// Ensure reserved series exist for scrapers/alerts before client ingest lands.
		renderErrorsTotal.WithLabelValues("_reserved").Add(0)
		offlineReplaysTotal.WithLabelValues("_reserved").Add(0)
	})
}

// IncInstanceAction increments lextures_content_tool_instances_total.
func IncInstanceAction(toolID, action string) {
	registerMetrics()
	instancesTotal.WithLabelValues(toolID, action).Inc()
}

// IncConfigValidationFailure increments validation failure counter.
func IncConfigValidationFailure(toolID string) {
	registerMetrics()
	configValidationFailures.WithLabelValues(toolID).Inc()
}

// IncInsert increments lextures_content_tool_insert_total{tool_id,surface}.
func IncInsert(toolID, surface string) {
	registerMetrics()
	if surface == "" {
		surface = "api"
	}
	insertTotal.WithLabelValues(toolID, surface).Inc()
}

// IncConfigSave increments lextures_content_tool_config_save_total{tool_id,outcome}.
func IncConfigSave(toolID, outcome string) {
	registerMetrics()
	if outcome == "" {
		outcome = "ok"
	}
	configSaveTotal.WithLabelValues(toolID, outcome).Inc()
}

// SetRegistrySizeGauge sets lextures_content_tool_registry_size.
func SetRegistrySizeGauge(n float64) {
	registerMetrics()
	registrySize.Set(n)
}

// RefreshKillSwitchMetric updates the kill-switch gauge from current state.
func RefreshKillSwitchMetric() {
	registerMetrics()
	if KillSwitchEngaged() {
		killSwitchEngaged.Set(1)
	} else {
		killSwitchEngaged.Set(0)
	}
}

// IncStateSave increments lextures_content_tool_state_saves_total{tool_id,outcome}.
func IncStateSave(toolID, outcome string) {
	registerMetrics()
	if outcome == "" {
		outcome = "ok"
	}
	stateSavesTotal.WithLabelValues(toolID, outcome).Inc()
}

// IncStateConflict increments lextures_content_tool_state_conflicts_total{tool_id}.
func IncStateConflict(toolID string) {
	registerMetrics()
	stateConflictsTotal.WithLabelValues(toolID).Inc()
}

// ObserveActionLatency records action latency seconds.
func ObserveActionLatency(toolID, action string, seconds float64) {
	registerMetrics()
	actionLatency.WithLabelValues(toolID, action).Observe(seconds)
}

