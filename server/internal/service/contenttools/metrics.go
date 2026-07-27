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

	resetsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "lextures",
		Name:      "content_tool_resets_total",
		Help:      "Content tool resets by tool_id, scope, and actor_role.",
	}, []string{"tool_id", "scope", "actor_role"})

	resetRowsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "lextures",
		Name:      "content_tool_reset_rows_total",
		Help:      "Content tool reset rows cleared by scope.",
	}, []string{"scope"})

	resetRestoresTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: "lextures",
		Name:      "content_tool_reset_restores_total",
		Help:      "Content tool reset snapshot restores.",
	})

	resetJobDuration = prometheus.NewHistogram(prometheus.HistogramOpts{
		Namespace: "lextures",
		Name:      "content_tool_reset_job_duration_seconds",
		Help:      "Async content tool reset job duration.",
		Buckets:   []float64{0.1, 0.5, 1, 2.5, 5, 10, 30, 60},
	})

	bridgeMessagesTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "lextures",
		Name:      "content_tool_bridge_messages_total",
		Help:      "Sandbox postMessage bridge outcomes by tool_id, type, and outcome.",
	}, []string{"tool_id", "type", "outcome"})

	migrationDocsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "lextures",
		Name:      "content_tool_migration_docs_total",
		Help:      "Content tool state migration document outcomes.",
	}, []string{"tool_id", "from", "to", "outcome"})

	breakerState = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "lextures",
		Name:      "content_tool_breaker_state",
		Help:      "1 when the per-tool circuit breaker is open, else 0.",
	}, []string{"tool_id"})

	bundleBytes = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "lextures",
		Name:      "content_tool_bundle_bytes",
		Help:      "Gzipped renderer bundle size in bytes by tool_id.",
	}, []string{"tool_id"})

	policyDenialsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "lextures",
		Name:      "content_tool_policy_denials_total",
		Help:      "Content tool policy denials by reason (CT.8).",
	}, []string{"reason"})

	moderationActionsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "lextures",
		Name:      "content_tool_moderation_actions_total",
		Help:      "Content tool moderation actions by action (CT.8).",
	}, []string{"action"})

	contentFilterFlagsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "lextures",
		Name:      "content_tool_content_filter_flags_total",
		Help:      "Content tool free-text filter flags by category (CT.8).",
	}, []string{"category"})

	crisisEscalationsTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: "lextures",
		Name:      "content_tool_crisis_escalations_total",
		Help:      "Content tool crisis-signal escalations (CT.8).",
	})

	a11yGateFailuresTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "lextures",
		Name:      "content_tool_a11y_gate_failures_total",
		Help:      "Content tool accessibility conformance gate failures by tool_id (CT.8).",
	}, []string{"tool_id"})

	askQuestionsTurnsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "lextures",
		Name:      "content_tool_ask_questions_turns_total",
		Help:      "Ask Questions turn outcomes by outcome (CT.10).",
	}, []string{"outcome"})

	askCitationsDroppedTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: "lextures",
		Name:      "content_tool_ask_citations_dropped_total",
		Help:      "Ask Questions citations dropped because they did not resolve to context segments (CT.10).",
	})

	inlineQuestionsSubmitsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "lextures",
		Name:      "content_tool_submits_total",
		Help:      "Content tool submit action outcomes by tool_id and outcome (CT.11).",
	}, []string{"tool_id", "outcome"})

	inlineQuestionsCorrectTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: "lextures",
		Name:      "content_tool_inline_questions_correct_total",
		Help:      "Inline Questions correct submissions (CT.11).",
	})

	inlineQuestionsAttemptHistogram = prometheus.NewHistogram(prometheus.HistogramOpts{
		Namespace: "lextures",
		Name:      "content_tool_inline_questions_attempt_count",
		Help:      "Inline Questions attempt count at submit time (CT.11).",
		Buckets:   []float64{1, 2, 3, 4, 5, 8, 12},
	})
)

func registerMetrics() {
	metricsOnce.Do(func() {
		prometheus.MustRegister(
			instancesTotal, configValidationFailures, insertTotal, configSaveTotal,
			registrySize, killSwitchEngaged,
			stateSavesTotal, stateConflictsTotal, actionLatency, renderErrorsTotal, offlineReplaysTotal,
			resetsTotal, resetRowsTotal, resetRestoresTotal, resetJobDuration,
			bridgeMessagesTotal, migrationDocsTotal, breakerState, bundleBytes,
			policyDenialsTotal, moderationActionsTotal, contentFilterFlagsTotal,
			crisisEscalationsTotal, a11yGateFailuresTotal,
			askQuestionsTurnsTotal, askCitationsDroppedTotal,
			inlineQuestionsSubmitsTotal, inlineQuestionsCorrectTotal, inlineQuestionsAttemptHistogram,
		)
		// Ensure reserved series exist for scrapers/alerts before client ingest lands.
		renderErrorsTotal.WithLabelValues("_reserved").Add(0)
		offlineReplaysTotal.WithLabelValues("_reserved").Add(0)
		bridgeMessagesTotal.WithLabelValues("_reserved", "_reserved", "_reserved").Add(0)
		migrationDocsTotal.WithLabelValues("_reserved", "0", "0", "_reserved").Add(0)
		policyDenialsTotal.WithLabelValues("_reserved").Add(0)
		moderationActionsTotal.WithLabelValues("_reserved").Add(0)
		contentFilterFlagsTotal.WithLabelValues("_reserved").Add(0)
		a11yGateFailuresTotal.WithLabelValues("_reserved").Add(0)
		askQuestionsTurnsTotal.WithLabelValues("_reserved").Add(0)
		inlineQuestionsSubmitsTotal.WithLabelValues("inline_questions", "_reserved").Add(0)
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

// IncResets increments lextures_content_tool_resets_total.
func IncResets(toolID, scope, actorRole string) {
	registerMetrics()
	if toolID == "" {
		toolID = "_unknown"
	}
	if actorRole == "" {
		actorRole = "instructor"
	}
	resetsTotal.WithLabelValues(toolID, scope, actorRole).Inc()
}

// IncResetRows increments lextures_content_tool_reset_rows_total by n.
func IncResetRows(scope string, n int) {
	registerMetrics()
	if n <= 0 {
		return
	}
	resetRowsTotal.WithLabelValues(scope).Add(float64(n))
}

// IncResetRestores increments lextures_content_tool_reset_restores_total.
func IncResetRestores() {
	registerMetrics()
	resetRestoresTotal.Inc()
}

// ObserveResetJobDuration records async reset job duration.
func ObserveResetJobDuration(seconds float64) {
	registerMetrics()
	resetJobDuration.Observe(seconds)
}

// IncBridgeMessage increments lextures_content_tool_bridge_messages_total.
func IncBridgeMessage(toolID, msgType, outcome string) {
	registerMetrics()
	if toolID == "" {
		toolID = "_unknown"
	}
	if msgType == "" {
		msgType = "_unknown"
	}
	if outcome == "" {
		outcome = "ok"
	}
	bridgeMessagesTotal.WithLabelValues(toolID, msgType, outcome).Inc()
}

// IncMigrationDocs increments lextures_content_tool_migration_docs_total.
func IncMigrationDocs(toolID, from, to, outcome string) {
	registerMetrics()
	if toolID == "" {
		toolID = "_unknown"
	}
	if outcome == "" {
		outcome = "ok"
	}
	migrationDocsTotal.WithLabelValues(toolID, from, to, outcome).Inc()
}

// SetBreakerStateGauge sets lextures_content_tool_breaker_state{tool_id}.
func SetBreakerStateGauge(toolID string, open float64) {
	registerMetrics()
	if toolID == "" {
		return
	}
	breakerState.WithLabelValues(toolID).Set(open)
}

// SetBundleBytesGauge sets lextures_content_tool_bundle_bytes{tool_id}.
func SetBundleBytesGauge(toolID string, bytes float64) {
	registerMetrics()
	if toolID == "" {
		return
	}
	bundleBytes.WithLabelValues(toolID).Set(bytes)
}

// IncPolicyDenial increments lextures_content_tool_policy_denials_total{reason}.
func IncPolicyDenial(reason string) {
	registerMetrics()
	if reason == "" {
		reason = "_unknown"
	}
	policyDenialsTotal.WithLabelValues(reason).Inc()
}

// IncModerationAction increments lextures_content_tool_moderation_actions_total{action}.
func IncModerationAction(action string) {
	registerMetrics()
	if action == "" {
		action = "_unknown"
	}
	moderationActionsTotal.WithLabelValues(action).Inc()
}

// IncContentFilterFlag increments lextures_content_tool_content_filter_flags_total{category}.
func IncContentFilterFlag(category string) {
	registerMetrics()
	if category == "" {
		category = "_unknown"
	}
	contentFilterFlagsTotal.WithLabelValues(category).Inc()
}

// IncCrisisEscalation increments lextures_content_tool_crisis_escalations_total.
func IncCrisisEscalation() {
	registerMetrics()
	crisisEscalationsTotal.Inc()
}

// IncA11yGateFailure increments lextures_content_tool_a11y_gate_failures_total{tool_id}.
func IncA11yGateFailure(toolID string) {
	registerMetrics()
	if toolID == "" {
		toolID = "_unknown"
	}
	a11yGateFailuresTotal.WithLabelValues(toolID).Inc()
}

// ObserveAskQuestionsTurn increments lextures_content_tool_ask_questions_turns_total{outcome}.
func ObserveAskQuestionsTurn(outcome string) {
	registerMetrics()
	if outcome == "" {
		outcome = "_unknown"
	}
	askQuestionsTurnsTotal.WithLabelValues(outcome).Inc()
}

// ObserveAskCitationsDropped increments lextures_content_tool_ask_citations_dropped_total.
func ObserveAskCitationsDropped(n int) {
	registerMetrics()
	if n <= 0 {
		return
	}
	askCitationsDroppedTotal.Add(float64(n))
}

// ObserveInlineQuestionsSubmit increments lextures_content_tool_submits_total{tool_id,outcome}.
func ObserveInlineQuestionsSubmit(outcome string) {
	registerMetrics()
	if outcome == "" {
		outcome = "_unknown"
	}
	inlineQuestionsSubmitsTotal.WithLabelValues("inline_questions", outcome).Inc()
}

// ObserveInlineQuestionsCorrect increments lextures_content_tool_inline_questions_correct_total.
func ObserveInlineQuestionsCorrect() {
	registerMetrics()
	inlineQuestionsCorrectTotal.Inc()
}

// ObserveInlineQuestionsAttemptCount observes attempt-count histogram.
func ObserveInlineQuestionsAttemptCount(n int) {
	registerMetrics()
	if n < 0 {
		return
	}
	inlineQuestionsAttemptHistogram.Observe(float64(n))
}

