package coursechecklist

import (
	"sync"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	apiMetricsOnce sync.Once

	apiRequests = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "lextures",
		Name:      "coursechecklist_api_requests_total",
		Help:      "Course checklist API requests by route and status class.",
	}, []string{"route", "status"})

	snapshotHits = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "lextures",
		Name:      "coursechecklist_snapshot_hits_total",
		Help:      "Checklist snapshot cache outcomes (hit|stale|miss).",
	}, []string{"result"})

	dismissals = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "lextures",
		Name:      "coursechecklist_dismissals_total",
		Help:      "Checklist dismissals by reason.",
	}, []string{"reason"})

	singleflightWaiters = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "lextures",
		Name:      "coursechecklist_singleflight_waiters",
		Help:      "Goroutines waiting on or holding checklist evaluation slots.",
	})
)

func registerAPIMetrics() {
	apiMetricsOnce.Do(func() {
		prometheus.MustRegister(apiRequests, snapshotHits, dismissals, singleflightWaiters)
	})
}

// ObserveAPIRequest increments the API request counter (route handlers).
func ObserveAPIRequest(route, status string) {
	registerAPIMetrics()
	if route == "" {
		route = "unknown"
	}
	if status == "" {
		status = "unknown"
	}
	apiRequests.WithLabelValues(route, status).Inc()
}

func observeSnapshotHit(result string) {
	registerAPIMetrics()
	if result == "" {
		result = "miss"
	}
	snapshotHits.WithLabelValues(result).Inc()
}

func observeDismissal(reason string) {
	registerAPIMetrics()
	if reason == "" {
		reason = "unspecified"
	}
	dismissals.WithLabelValues(reason).Inc()
}

func setSingleflightWaiters(n float64) {
	registerAPIMetrics()
	singleflightWaiters.Set(n)
}

// SnapshotHitsCounter exposes the snapshot-hit counter for tests.
func SnapshotHitsCounter() *prometheus.CounterVec {
	registerAPIMetrics()
	return snapshotHits
}
