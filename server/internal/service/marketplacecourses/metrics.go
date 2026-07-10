package marketplacecourses

import (
	"fmt"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	metricsOnce       sync.Once
	provisionTotal    *prometheus.CounterVec
	provisionDuration prometheus.Histogram
	contentSyncTotal  *prometheus.CounterVec
	contentSyncDur    prometheus.Histogram
)

func initMetrics() {
	provisionTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "lextures",
		Name:      "marketplace_course_provision_total",
		Help:      "Official marketplace course provisioning runs by result and slug.",
	}, []string{"slug", "result"})
	provisionDuration = prometheus.NewHistogram(prometheus.HistogramOpts{
		Namespace: "lextures",
		Name:      "marketplace_course_provision_duration_seconds",
		Help:      "Official marketplace course provisioning duration in seconds.",
		Buckets:   []float64{0.001, 0.005, 0.01, 0.05, 0.1, 0.25, 0.5, 1, 2, 5, 10},
	})
	contentSyncTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "lextures",
		Name:      "marketplace_course_content_sync_total",
		Help:      "Official marketplace course content sync runs by result and slug.",
	}, []string{"slug", "result"})
	contentSyncDur = prometheus.NewHistogram(prometheus.HistogramOpts{
		Namespace: "lextures",
		Name:      "marketplace_course_content_sync_duration_seconds",
		Help:      "Official marketplace course content sync duration in seconds.",
		Buckets:   []float64{0.001, 0.005, 0.01, 0.05, 0.1, 0.25, 0.5, 1, 2, 5},
	})
}

// RegisterMetrics registers marketplace course metrics with reg (no-op when reg is nil).
func RegisterMetrics(reg prometheus.Registerer) {
	if reg == nil {
		return
	}
	metricsOnce.Do(func() {
		initMetrics()
		reg.MustRegister(provisionTotal, provisionDuration, contentSyncTotal, contentSyncDur)
	})
}

func recordProvision(slug, result string, started time.Time) {
	if provisionTotal == nil {
		return
	}
	provisionTotal.WithLabelValues(slug, result).Inc()
	provisionDuration.Observe(time.Since(started).Seconds())
}

func recordContentSync(slug, result string, started time.Time) {
	if contentSyncTotal == nil {
		return
	}
	contentSyncTotal.WithLabelValues(slug, result).Inc()
	contentSyncDur.Observe(time.Since(started).Seconds())
}

// FormatProvisionSummary returns a human-readable provision summary line.
func FormatProvisionSummary(slug string, created, modules, pages, assignments, quizzes, skipped int) string {
	return fmt.Sprintf(
		"slug=%s created=%d modules=%d pages=%d assignments=%d quizzes=%d skipped_items=%d",
		slug, created, modules, pages, assignments, quizzes, skipped,
	)
}
