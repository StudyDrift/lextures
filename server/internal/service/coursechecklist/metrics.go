package coursechecklist

import (
	"sync"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	metricsOnce sync.Once

	evaluateDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "lextures",
		Name:      "coursechecklist_evaluate_duration_seconds",
		Help:      "Course checklist evaluation duration by mode (full|single).",
		Buckets:   []float64{0.05, 0.1, 0.12, 0.25, 0.4, 0.5, 0.9, 1, 2.5, 5},
	}, []string{"mode"})

	ruleDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "lextures",
		Name:      "coursechecklist_rule_duration_seconds",
		Help:      "Per-rule checklist evaluation duration by item_id.",
		Buckets:   []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1},
	}, []string{"item_id"})

	ruleErrors = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "lextures",
		Name:      "coursechecklist_rule_errors_total",
		Help:      "Checklist rule failures by item_id and kind (error|panic|timeout).",
	}, []string{"item_id", "kind"})

	snapshotQueryDuration = prometheus.NewHistogram(prometheus.HistogramOpts{
		Namespace: "lextures",
		Name:      "coursechecklist_snapshot_query_duration_seconds",
		Help:      "Course checklist snapshot load duration.",
		Buckets:   []float64{0.01, 0.05, 0.1, 0.25, 0.4, 0.5, 0.9, 1, 2.5, 5},
	})

	// CC.10 FR-14: per-item status counts aggregated across evaluations (no course label).
	// Accommodations rules are excluded (FR-16 / AC-8).
	itemStatusTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "lextures",
		Name:      "coursechecklist_item_status_total",
		Help:      "Checklist item evaluation status counts by item_id and status (aggregated; no course id).",
	}, []string{"item_id", "status"})
)

// accommodationItemIDs must never appear on analytics counters that could carry counts (FR-16).
var accommodationItemIDs = map[ItemID]struct{}{
	ItemAccommodationsHonored:  {},
	ItemAccommodationsReviewed: {},
}

func registerMetrics() {
	metricsOnce.Do(func() {
		prometheus.MustRegister(evaluateDuration, ruleDuration, ruleErrors, snapshotQueryDuration, itemStatusTotal)
	})
}

// observeItemStatusCounts increments aggregated pass/todo metrics (CC.10 FR-14).
func observeItemStatusCounts(findings []ItemResult) {
	registerMetrics()
	for _, fr := range findings {
		if _, skip := accommodationItemIDs[fr.ID]; skip {
			continue
		}
		status := string(fr.Finding.Status)
		if status == "" {
			status = "unknown"
		}
		itemStatusTotal.WithLabelValues(string(fr.ID), status).Inc()
	}
}

func observeEvaluateDuration(mode string, seconds float64) {
	registerMetrics()
	if mode == "" {
		mode = "full"
	}
	evaluateDuration.WithLabelValues(mode).Observe(seconds)
}

func observeRuleDuration(itemID ItemID, seconds float64) {
	registerMetrics()
	ruleDuration.WithLabelValues(string(itemID)).Observe(seconds)
}

func incRuleError(itemID ItemID, kind string) {
	registerMetrics()
	if kind == "" {
		kind = "error"
	}
	ruleErrors.WithLabelValues(string(itemID), kind).Inc()
}

func observeSnapshotQueryDuration(seconds float64) {
	registerMetrics()
	snapshotQueryDuration.Observe(seconds)
}

// ruleErrorsCounter exposes the rule-error counter for tests in this package.
func ruleErrorsCounter() *prometheus.CounterVec {
	registerMetrics()
	return ruleErrors
}
