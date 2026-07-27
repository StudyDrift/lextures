package toolmarket

import (
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	metricsOnce sync.Once

	installsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "lextures_content_tool_marketplace_installs_total",
		Help: "Content tool marketplace install lifecycle actions",
	}, []string{"tool_id", "action"})

	reviewQueueDepth = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "lextures_content_tool_marketplace_review_queue_depth",
		Help: "Depth of the human review queue for third-party tools",
	})

	thirdPartyErrors = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "lextures_thirdparty_tool_errors_total",
		Help: "Errors from third-party content tools",
	}, []string{"tool_id"})

	bundleLoadSeconds = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "lextures_thirdparty_bundle_load_seconds",
		Help:    "Third-party tool bundle load latency",
		Buckets: prometheus.DefBuckets,
	}, []string{"tool_id"})

	queueMu    sync.Mutex
	queueDepth float64
)

// IncInstall increments install action counter.
func IncInstall(toolID, action string) {
	metricsOnce.Do(func() {})
	installsTotal.WithLabelValues(toolID, action).Inc()
}

// IncReviewQueue adjusts review queue depth gauge.
func IncReviewQueue(delta float64) {
	queueMu.Lock()
	defer queueMu.Unlock()
	queueDepth += delta
	if queueDepth < 0 {
		queueDepth = 0
	}
	reviewQueueDepth.Set(queueDepth)
}

// IncThirdPartyError increments third-party error counter.
func IncThirdPartyError(toolID string) {
	thirdPartyErrors.WithLabelValues(toolID).Inc()
}

// ObserveBundleLoad records bundle load seconds.
func ObserveBundleLoad(toolID string, seconds float64) {
	bundleLoadSeconds.WithLabelValues(toolID).Observe(seconds)
}
