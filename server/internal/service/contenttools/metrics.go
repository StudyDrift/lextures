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
)

func registerMetrics() {
	metricsOnce.Do(func() {
		prometheus.MustRegister(instancesTotal, configValidationFailures, registrySize, killSwitchEngaged)
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
