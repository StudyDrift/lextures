package telemetry

import (
	"strconv"

	"github.com/prometheus/client_golang/prometheus"
)

func (m *Metrics) registerAPIErrorMetrics(reg *prometheus.Registry) {
	m.mappedAPIErrorsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Name:      "mapped_api_errors_total",
		Help:      "API errors mapped by the handler toolkit, labelled by code and route class (TD.7).",
	}, []string{"code", "route_class", "status"})
	reg.MustRegister(m.mappedAPIErrorsTotal)
}

// RecordMappedAPIError increments the toolkit error mapper counter.
func (m *Metrics) RecordMappedAPIError(code, routeClass string, status int) {
	if m == nil || m.mappedAPIErrorsTotal == nil {
		return
	}
	if code == "" {
		code = "UNKNOWN"
	}
	if routeClass == "" {
		routeClass = "unknown"
	}
	m.mappedAPIErrorsTotal.WithLabelValues(code, routeClass, strconv.Itoa(status)).Inc()
}
