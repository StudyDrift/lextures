package adaptivecontent

import (
	"sync"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	metricsOnce          sync.Once
	coursesEnabled       prometheus.Gauge
	settingsUpdated      prometheus.Counter
	killSwitchEngagedG   prometheus.Gauge
	profileComputeMs     prometheus.Histogram
	profileEmphasisTotal *prometheus.CounterVec
	distinctSignatures   prometheus.Gauge
	// AC.3 generation
	generateMs         prometheus.Histogram
	fidelityScoreHist  prometheus.Histogram
	generatedTotal     *prometheus.CounterVec
	rejectedFidelity   prometheus.Counter
	rejectedSafety     prometheus.Counter
	cacheHitTotal      prometheus.Counter
	// AC.4 pipeline
	cacheMissTotal     prometheus.Counter
	queueDepthG        prometheus.Gauge
	inflightG          prometheus.Gauge
	jobRetryTotal      prometheus.Counter
	budgetExhaustedTot prometheus.Counter
	jobLatencyMs       prometheus.Histogram
	// AC.5 review
	approvedTotal   prometheus.Counter
	rejectedTotal   prometheus.Counter
	editedTotal     prometheus.Counter
	revokedTotal    prometheus.Counter
	autoServedTotal prometheus.Counter
	timeInQueueMs   prometheus.Histogram
	// AC.6 student runtime
	servedVariantTotal   prometheus.Counter
	servedBaseTotal      prometheus.Counter
	servedHoldoutTotal   prometheus.Counter
	servedFallbackTotal  prometheus.Counter
	viewOriginalClicks   prometheus.Counter
	optoutTotal          prometheus.Counter
	serveLatencyMs       prometheus.Histogram
	// AC.7 effectiveness
	outcomesRecordedTotal   prometheus.Counter
	verdictRegressingTotal  prometheus.Counter
	unitMeanLiftG           prometheus.Gauge
	treatmentMinusHoldoutG  prometheus.Gauge
	outcomeRecordMs         prometheus.Histogram
)

func initMetrics() {
	coursesEnabled = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "lextures",
		Name:      "adaptive_content_courses_enabled",
		Help:      "Number of courses with adaptive_content_enabled = true (refreshed on toggle).",
	})
	settingsUpdated = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: "lextures",
		Name:      "adaptive_content_settings_updated_total",
		Help:      "Count of adaptive content settings PUT successes.",
	})
	killSwitchEngagedG = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "lextures",
		Name:      "adaptive_content_kill_switch_engaged",
		Help:      "1 when ADAPTIVE_CONTENT_KILL_SWITCH is engaged, else 0.",
	})
	profileComputeMs = prometheus.NewHistogram(prometheus.HistogramOpts{
		Namespace: "lextures",
		Name:      "adaptive_content_profile_compute_ms",
		Help:      "Wall time in milliseconds to compute and upsert an adaptation profile (AC.2).",
		Buckets:   []float64{1, 5, 10, 25, 50, 100, 150, 250, 500, 1000},
	})
	profileEmphasisTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "lextures",
		Name:      "adaptive_content_profile_emphasis_total",
		Help:      "Count of profiles computed by emphasis_mode (AC.2).",
	}, []string{"emphasis_mode"})
	distinctSignatures = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "lextures",
		Name:      "adaptive_content_distinct_signatures_per_unit",
		Help:      "Last observed distinct profile_signature count for a unit (updated on cohort reads).",
	})
	generateMs = prometheus.NewHistogram(prometheus.HistogramOpts{
		Namespace: "lextures",
		Name:      "adaptive_content_generate_ms",
		Help:      "Wall time in milliseconds for adaptive content variant generation (AC.3).",
		Buckets:   []float64{50, 100, 250, 500, 1000, 2000, 4000, 8000, 15000, 30000},
	})
	fidelityScoreHist = prometheus.NewHistogram(prometheus.HistogramOpts{
		Namespace: "lextures",
		Name:      "adaptive_content_fidelity_score",
		Help:      "Fidelity score distribution for generated variants (AC.3).",
		Buckets:   []float64{0, 0.25, 0.5, 0.7, 0.85, 0.9, 0.95, 1.0},
	})
	generatedTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "lextures",
		Name:      "adaptive_content_generated_total",
		Help:      "Count of adaptive content generation outcomes by result (AC.3).",
	}, []string{"result"})
	rejectedFidelity = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: "lextures",
		Name:      "adaptive_content_rejected_fidelity_total",
		Help:      "Count of variants rejected by the fidelity gate (AC.3).",
	})
	rejectedSafety = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: "lextures",
		Name:      "adaptive_content_rejected_safety_total",
		Help:      "Count of variants rejected by the safety gate (AC.3).",
	})
	cacheHitTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: "lextures",
		Name:      "adaptive_content_cache_hit_total",
		Help:      "Count of content_variants cache hits (AC.3/AC.4).",
	})
	cacheMissTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: "lextures",
		Name:      "adaptive_content_cache_miss_total",
		Help:      "Count of content_variants cache misses (AC.4).",
	})
	queueDepthG = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "lextures",
		Name:      "adaptive_content_queue_depth",
		Help:      "Pending adaptive content generation jobs (AC.4).",
	})
	inflightG = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "lextures",
		Name:      "adaptive_content_inflight",
		Help:      "In-flight (generating) adaptive content jobs (AC.4).",
	})
	jobRetryTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: "lextures",
		Name:      "adaptive_content_job_retry_total",
		Help:      "Count of adaptive content job retries / requeues (AC.4).",
	})
	budgetExhaustedTot = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: "lextures",
		Name:      "adaptive_content_budget_exhausted_total",
		Help:      "Count of generations skipped due to monthly token budget (AC.4).",
	})
	jobLatencyMs = prometheus.NewHistogram(prometheus.HistogramOpts{
		Namespace: "lextures",
		Name:      "adaptive_content_job_latency_ms",
		Help:      "End-to-end adaptive content job processing latency in ms (AC.4).",
		Buckets:   []float64{50, 100, 250, 500, 1000, 2000, 4000, 8000, 15000, 30000, 60000},
	})
	approvedTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: "lextures",
		Name:      "adaptive_content_approved_total",
		Help:      "Count of adaptive content variants approved by instructors (AC.5).",
	})
	rejectedTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: "lextures",
		Name:      "adaptive_content_rejected_total",
		Help:      "Count of adaptive content variants rejected by instructors (AC.5).",
	})
	editedTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: "lextures",
		Name:      "adaptive_content_edited_total",
		Help:      "Count of adaptive content variants edit-and-approved by instructors (AC.5).",
	})
	revokedTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: "lextures",
		Name:      "adaptive_content_revoked_total",
		Help:      "Count of adaptive content variants revoked (superseded) by instructors (AC.5).",
	})
	autoServedTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: "lextures",
		Name:      "adaptive_content_auto_served_total",
		Help:      "Count of adaptive content variants auto-served after gates (AC.5 observability).",
	})
	timeInQueueMs = prometheus.NewHistogram(prometheus.HistogramOpts{
		Namespace: "lextures",
		Name:      "adaptive_content_time_in_queue_ms",
		Help:      "Time from variant creation to instructor review decision in ms (AC.5).",
		Buckets:   []float64{1000, 5000, 15000, 60000, 300000, 900000, 3600000, 86400000},
	})
	servedVariantTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: "lextures",
		Name:      "adaptive_content_served_variant_total",
		Help:      "Count of student content views served an adapted variant (AC.6).",
	})
	servedBaseTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: "lextures",
		Name:      "adaptive_content_served_base_total",
		Help:      "Count of student content views served base content intentionally (AC.6).",
	})
	servedHoldoutTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: "lextures",
		Name:      "adaptive_content_served_holdout_total",
		Help:      "Count of student content views in the holdout/control group (AC.6).",
	})
	servedFallbackTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: "lextures",
		Name:      "adaptive_content_served_fallback_total",
		Help:      "Count of student content views that fell back to base (miss/optout/error) (AC.6).",
	})
	viewOriginalClicks = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: "lextures",
		Name:      "adaptive_content_view_original_clicks_total",
		Help:      "Count of student View original toggles (AC.6).",
	})
	optoutTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: "lextures",
		Name:      "adaptive_content_optout_total",
		Help:      "Count of adaptive content views where student opt-out was honored (AC.6).",
	})
	serveLatencyMs = prometheus.NewHistogram(prometheus.HistogramOpts{
		Namespace: "lextures",
		Name:      "adaptive_content_serve_latency_ms",
		Help:      "Wall time in milliseconds for adaptive content serving resolution (AC.6).",
		Buckets:   []float64{1, 2, 5, 10, 15, 20, 30, 50, 100, 250, 500},
	})
	outcomesRecordedTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: "lextures",
		Name:      "adaptive_content_outcomes_recorded_total",
		Help:      "Count of adaptation_outcomes upserts after post-assessment submit (AC.7).",
	})
	verdictRegressingTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: "lextures",
		Name:      "adaptive_content_verdict_regressing_total",
		Help:      "Count of times a unit verdict transitions to regressing (AC.7).",
	})
	unitMeanLiftG = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "lextures",
		Name:      "adaptive_content_unit_mean_lift",
		Help:      "Last refreshed mean lift for the treatment arm of a unit (AC.7).",
	})
	treatmentMinusHoldoutG = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "lextures",
		Name:      "adaptive_content_treatment_minus_holdout",
		Help:      "Last refreshed treatment−holdout mean lift difference (AC.7).",
	})
	outcomeRecordMs = prometheus.NewHistogram(prometheus.HistogramOpts{
		Namespace: "lextures",
		Name:      "adaptive_content_outcome_record_ms",
		Help:      "Wall time in milliseconds to record a post-assessment outcome (AC.7).",
		Buckets:   []float64{1, 5, 10, 25, 50, 100, 150, 250, 500},
	})
	prometheus.MustRegister(
		coursesEnabled, settingsUpdated, killSwitchEngagedG,
		profileComputeMs, profileEmphasisTotal, distinctSignatures,
		generateMs, fidelityScoreHist, generatedTotal, rejectedFidelity, rejectedSafety, cacheHitTotal,
		cacheMissTotal, queueDepthG, inflightG, jobRetryTotal, budgetExhaustedTot, jobLatencyMs,
		approvedTotal, rejectedTotal, editedTotal, revokedTotal, autoServedTotal, timeInQueueMs,
		servedVariantTotal, servedBaseTotal, servedHoldoutTotal, servedFallbackTotal,
		viewOriginalClicks, optoutTotal, serveLatencyMs,
		outcomesRecordedTotal, verdictRegressingTotal, unitMeanLiftG, treatmentMinusHoldoutG, outcomeRecordMs,
	)
	if KillSwitchEngaged() {
		killSwitchEngagedG.Set(1)
	} else {
		killSwitchEngagedG.Set(0)
	}
}

func ensureMetrics() {
	metricsOnce.Do(initMetrics)
}

// RefreshKillSwitchMetric exports the current kill-switch state as a gauge.
func RefreshKillSwitchMetric() {
	ensureMetrics()
	if KillSwitchEngaged() {
		killSwitchEngagedG.Set(1)
	} else {
		killSwitchEngagedG.Set(0)
	}
}

// SetCoursesEnabledGauge sets the courses-enabled gauge.
func SetCoursesEnabledGauge(n float64) {
	ensureMetrics()
	coursesEnabled.Set(n)
}

// IncSettingsUpdated increments the settings-updated counter.
func IncSettingsUpdated() {
	ensureMetrics()
	settingsUpdated.Inc()
}

// ObserveProfileCompute records profile compute latency in milliseconds.
func ObserveProfileCompute(ms float64) {
	ensureMetrics()
	profileComputeMs.Observe(ms)
}

// IncProfileEmphasis increments the counter for the given emphasis_mode.
func IncProfileEmphasis(mode string) {
	ensureMetrics()
	if mode == "" {
		mode = "unknown"
	}
	profileEmphasisTotal.WithLabelValues(mode).Inc()
}

// SetDistinctSignaturesPerUnit sets the gauge for distinct signatures (last unit observed).
func SetDistinctSignaturesPerUnit(n float64) {
	ensureMetrics()
	distinctSignatures.Set(n)
}

// ObserveGenerate records generation latency in milliseconds.
func ObserveGenerate(ms float64) {
	ensureMetrics()
	generateMs.Observe(ms)
}

// ObserveFidelity records a fidelity score observation.
func ObserveFidelity(score float64) {
	ensureMetrics()
	fidelityScoreHist.Observe(score)
}

// IncGenerated increments the generation outcome counter.
func IncGenerated(result string) {
	ensureMetrics()
	if result == "" {
		result = "unknown"
	}
	generatedTotal.WithLabelValues(result).Inc()
}

// IncRejectedFidelity increments the fidelity-reject counter.
func IncRejectedFidelity() {
	ensureMetrics()
	rejectedFidelity.Inc()
}

// IncRejectedSafety increments the safety-reject counter.
func IncRejectedSafety() {
	ensureMetrics()
	rejectedSafety.Inc()
}

// IncCacheHit increments the variant cache-hit counter.
func IncCacheHit() {
	ensureMetrics()
	cacheHitTotal.Inc()
}

// IncCacheMiss increments the variant cache-miss counter.
func IncCacheMiss() {
	ensureMetrics()
	cacheMissTotal.Inc()
}

// SetQueueDepth sets the pending job gauge.
func SetQueueDepth(n float64) {
	ensureMetrics()
	queueDepthG.Set(n)
}

// SetInflight sets the generating job gauge.
func SetInflight(n float64) {
	ensureMetrics()
	inflightG.Set(n)
}

// IncJobRetry increments the job retry counter.
func IncJobRetry() {
	ensureMetrics()
	jobRetryTotal.Inc()
}

// IncBudgetExhausted increments the budget-exhausted counter.
func IncBudgetExhausted() {
	ensureMetrics()
	budgetExhaustedTot.Inc()
}

// ObserveJobLatency records end-to-end job processing latency.
func ObserveJobLatency(ms float64) {
	ensureMetrics()
	jobLatencyMs.Observe(ms)
}

// IncApproved increments the instructor-approved counter.
func IncApproved() {
	ensureMetrics()
	approvedTotal.Inc()
}

// IncRejected increments the instructor-rejected counter.
func IncRejected() {
	ensureMetrics()
	rejectedTotal.Inc()
}

// IncEdited increments the edit-and-approve counter.
func IncEdited() {
	ensureMetrics()
	editedTotal.Inc()
}

// IncRevoked increments the revoke counter.
func IncRevoked() {
	ensureMetrics()
	revokedTotal.Inc()
}

// IncAutoServed increments the auto-serve counter.
func IncAutoServed() {
	ensureMetrics()
	autoServedTotal.Inc()
}

// ObserveTimeInQueue records time from creation to review decision.
func ObserveTimeInQueue(ms float64) {
	ensureMetrics()
	if ms > 0 {
		timeInQueueMs.Observe(ms)
	}
}

// IncServedVariant increments the adapted-variant serve counter (AC.6).
func IncServedVariant() {
	ensureMetrics()
	servedVariantTotal.Inc()
}

// IncServedBase increments the intentional base-serve counter (AC.6).
func IncServedBase() {
	ensureMetrics()
	servedBaseTotal.Inc()
}

// IncServedHoldout increments the holdout/control serve counter (AC.6).
func IncServedHoldout() {
	ensureMetrics()
	servedHoldoutTotal.Inc()
}

// IncServedFallback increments the fallback-to-base serve counter (AC.6).
func IncServedFallback() {
	ensureMetrics()
	servedFallbackTotal.Inc()
}

// IncViewOriginalClicks increments the View original click counter (AC.6).
func IncViewOriginalClicks() {
	ensureMetrics()
	viewOriginalClicks.Inc()
}

// IncOptout increments the opt-out honored counter (AC.6).
func IncOptout() {
	ensureMetrics()
	optoutTotal.Inc()
}

// ObserveServeLatency records serve-resolution latency in milliseconds (AC.6).
func ObserveServeLatency(ms float64) {
	ensureMetrics()
	if ms >= 0 {
		serveLatencyMs.Observe(ms)
	}
}

// IncOutcomesRecorded increments the post-assessment outcome counter (AC.7).
func IncOutcomesRecorded() {
	ensureMetrics()
	outcomesRecordedTotal.Inc()
}

// IncVerdictRegressing increments the regressing-verdict counter (AC.7).
func IncVerdictRegressing() {
	ensureMetrics()
	verdictRegressingTotal.Inc()
}

// SetUnitMeanLift sets the treatment mean-lift gauge (AC.7).
func SetUnitMeanLift(v float64) {
	ensureMetrics()
	unitMeanLiftG.Set(v)
}

// SetTreatmentMinusHoldout sets the treatment−holdout difference gauge (AC.7).
func SetTreatmentMinusHoldout(v float64) {
	ensureMetrics()
	treatmentMinusHoldoutG.Set(v)
}

// ObserveOutcomeRecord records post-assessment outcome upsert latency (AC.7).
func ObserveOutcomeRecord(ms float64) {
	ensureMetrics()
	if ms >= 0 {
		outcomeRecordMs.Observe(ms)
	}
}
