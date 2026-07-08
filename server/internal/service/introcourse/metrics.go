package introcourse

import (
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	metricsOnce          sync.Once
	provisionTotal       *prometheus.CounterVec
	provisionDuration    prometheus.Histogram
	coursePresent        prometheus.Gauge
	enrollTotal          *prometheus.CounterVec
	backfillProgress     prometheus.Gauge
	backfillRemaining    prometheus.Gauge
	contentSyncTotal      *prometheus.CounterVec
	contentSyncDuration   prometheus.Histogram
	contentVersionGauge   prometheus.Gauge
	autogradeTotal        *prometheus.CounterVec
	gradeWriteTotal       prometheus.Counter
	graderAgentFallback   prometheus.Counter
	completionTotal       prometheus.Counter
	progressRecomputeTotal prometheus.Counter
	credentialIssuedTotal prometheus.Counter
	completionRateGauge   prometheus.Gauge
	adminActionTotal      *prometheus.CounterVec
)

func initMetrics() {
	provisionTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "lextures",
		Name:      "intro_course_provision_total",
		Help:      "Intro course provisioning runs by result.",
	}, []string{"result"})
	provisionDuration = prometheus.NewHistogram(prometheus.HistogramOpts{
		Namespace: "lextures",
		Name:      "intro_course_provision_duration_seconds",
		Help:      "Intro course provisioning duration in seconds.",
		Buckets:   []float64{0.001, 0.005, 0.01, 0.05, 0.1, 0.25, 0.5, 1},
	})
	coursePresent = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "lextures",
		Name:      "intro_course_present",
		Help:      "Whether the canonical intro course row exists (1) or not (0).",
	})
	enrollTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "lextures",
		Name:      "intro_course_enroll_total",
		Help:      "Intro course student enrollments by creation path and result.",
	}, []string{"path", "result"})
	backfillProgress = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "lextures",
		Name:      "intro_course_backfill_progress",
		Help:      "Whether intro course backfill has started or completed (1) or not (0).",
	})
	backfillRemaining = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "lextures",
		Name:      "intro_course_backfill_remaining",
		Help:      "Eligible users not yet enrolled in the intro course.",
	})
	contentSyncTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "lextures",
		Name:      "intro_course_content_sync_total",
		Help:      "Intro course curriculum content sync runs by result.",
	}, []string{"result"})
	contentSyncDuration = prometheus.NewHistogram(prometheus.HistogramOpts{
		Namespace: "lextures",
		Name:      "intro_course_content_sync_duration_seconds",
		Help:      "Intro course content sync duration in seconds.",
		Buckets:   []float64{0.001, 0.005, 0.01, 0.05, 0.1, 0.25, 0.5, 1, 2},
	})
	contentVersionGauge = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "lextures",
		Name:      "intro_course_content_version",
		Help:      "Deployed intro course curriculum content version.",
	})
	autogradeTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "lextures",
		Name:      "intro_course_autograde_total",
		Help:      "Intro course auto-grade attempts by item type and result.",
	}, []string{"item_type", "result"})
	gradeWriteTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: "lextures",
		Name:      "intro_course_grade_write_total",
		Help:      "Intro course gradebook writes from auto-grading.",
	})
	graderAgentFallback = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: "lextures",
		Name:      "intro_course_grader_agent_fallback_total",
		Help:      "Intro course grader-agent failures where completion credit was already awarded.",
	})
	completionTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: "lextures",
		Name:      "intro_course_completion_total",
		Help:      "Intro course completions recorded (set-once per learner).",
	})
	progressRecomputeTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: "lextures",
		Name:      "intro_course_progress_recompute_total",
		Help:      "Intro course progress reads/recomputes.",
	})
	credentialIssuedTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: "lextures",
		Name:      "intro_course_credential_issued_total",
		Help:      "Intro course completion credentials issued.",
	})
	completionRateGauge = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "lextures",
		Name:      "intro_course_completion_rate",
		Help:      "Ratio of completers to enrolled students (0–1).",
	})
	adminActionTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "lextures",
		Name:      "intro_course_admin_action_total",
		Help:      "Intro course admin actions by type (IC08).",
	}, []string{"action"})
}

// RegisterMetrics registers intro course metrics with reg (no-op when reg is nil).
func RegisterMetrics(reg prometheus.Registerer) {
	if reg == nil {
		return
	}
	metricsOnce.Do(func() {
		initMetrics()
		reg.MustRegister(
			provisionTotal, provisionDuration, coursePresent, enrollTotal,
			backfillProgress, backfillRemaining,
			contentSyncTotal, contentSyncDuration, contentVersionGauge,
			autogradeTotal, gradeWriteTotal, graderAgentFallback,
			completionTotal, progressRecomputeTotal, credentialIssuedTotal, completionRateGauge,
			adminActionTotal,
		)
	})
}

func recordProvision(result string, started time.Time) {
	if provisionTotal == nil {
		return
	}
	provisionTotal.WithLabelValues(result).Inc()
	provisionDuration.Observe(time.Since(started).Seconds())
}

func setCoursePresent(present bool) {
	if coursePresent == nil {
		return
	}
	if present {
		coursePresent.Set(1)
	} else {
		coursePresent.Set(0)
	}
}

func recordEnroll(path, result string) {
	if enrollTotal == nil {
		return
	}
	enrollTotal.WithLabelValues(path, result).Inc()
}

func setBackfillProgress(v float64) {
	if backfillProgress == nil {
		return
	}
	backfillProgress.Set(v)
}

func setBackfillRemaining(v float64) {
	if backfillRemaining == nil {
		return
	}
	backfillRemaining.Set(v)
}

func recordContentSync(result string, started time.Time) {
	if contentSyncTotal == nil {
		return
	}
	contentSyncTotal.WithLabelValues(result).Inc()
	contentSyncDuration.Observe(time.Since(started).Seconds())
}

func setContentVersionGauge(v int) {
	if contentVersionGauge == nil {
		return
	}
	contentVersionGauge.Set(float64(v))
}

func recordAutograde(itemType, result string) {
	if autogradeTotal == nil {
		return
	}
	autogradeTotal.WithLabelValues(itemType, result).Inc()
}

func recordGradeWrite() {
	if gradeWriteTotal == nil {
		return
	}
	gradeWriteTotal.Inc()
}

// RecordGraderAgentFallback increments the fallback counter when grader-agent fails after completion credit.
func RecordGraderAgentFallback() {
	if graderAgentFallback == nil {
		return
	}
	graderAgentFallback.Inc()
}

func recordCompletion() {
	if completionTotal == nil {
		return
	}
	completionTotal.Inc()
}

func recordProgressRecompute() {
	if progressRecomputeTotal == nil {
		return
	}
	progressRecomputeTotal.Inc()
}

func recordCredentialIssued() {
	if credentialIssuedTotal == nil {
		return
	}
	credentialIssuedTotal.Inc()
}

func setCompletionRate(rate float64) {
	if completionRateGauge == nil {
		return
	}
	completionRateGauge.Set(rate)
}

// RecordAdminAction increments intro_course_admin_action_total{action}.
func RecordAdminAction(action string) {
	if adminActionTotal == nil {
		return
	}
	adminActionTotal.WithLabelValues(action).Inc()
}