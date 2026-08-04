package linkhealth

import (
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	metricsOnce sync.Once

	linkcheckDuration prometheus.Histogram
	urlsTotal         *prometheus.CounterVec
	blockedTotal      *prometheus.CounterVec
)

func ensureMetrics() {
	metricsOnce.Do(func() {
		linkcheckDuration = promauto.NewHistogram(prometheus.HistogramOpts{
			Name:    "coursechecklist_linkcheck_duration_seconds",
			Help:    "Wall-clock duration of a course link-health check job.",
			Buckets: prometheus.DefBuckets,
		})
		urlsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "coursechecklist_linkcheck_urls_total",
			Help: "Link-health URL outcomes.",
		}, []string{"result"})
		blockedTotal = promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "coursechecklist_linkcheck_blocked_total",
			Help: "Link-health URLs blocked before fetch.",
		}, []string{"reason"})
	})
}

func ObserveDuration(seconds float64) {
	ensureMetrics()
	linkcheckDuration.Observe(seconds)
}

func incURLResult(r ResultCode) {
	ensureMetrics()
	urlsTotal.WithLabelValues(string(r)).Inc()
}

func incBlocked(reason BlockedReason) {
	ensureMetrics()
	if reason == "" {
		return
	}
	blockedTotal.WithLabelValues(string(reason)).Inc()
}
