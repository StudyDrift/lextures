package contenttools

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"

	ctrepo "github.com/lextures/lextures/server/internal/repos/contenttools"
	"github.com/lextures/lextures/server/internal/service/aiprovider"
	ctctx "github.com/lextures/lextures/server/internal/service/contenttools/context"
	"github.com/lextures/lextures/server/internal/service/contenttools/tools/explain_it_back"
)

var (
	explainItBackMetricsOnce sync.Once

	explainItBackSubmitsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "lextures",
		Name:      "content_tool_explain_it_back_submits_total",
		Help:      "Explain It Back submit outcomes by outcome (CT.20).",
	}, []string{"outcome"})

	explainItBackSchemaFailuresTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: "lextures",
		Name:      "content_tool_explain_it_back_schema_failures_total",
		Help:      "Explain It Back structured-output validation failures (CT.20).",
	})

	explainItBackRevisionsTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: "lextures",
		Name:      "content_tool_explain_it_back_revisions_total",
		Help:      "Explain It Back second-or-later attempts after feedback (CT.20).",
	})
)

func registerExplainItBackMetrics() {
	explainItBackMetricsOnce.Do(func() {
		prometheus.MustRegister(
			explainItBackSubmitsTotal,
			explainItBackSchemaFailuresTotal,
			explainItBackRevisionsTotal,
		)
		explainItBackSubmitsTotal.WithLabelValues("_reserved").Add(0)
	})
}

// ObserveExplainItBackSubmit increments submit outcome counter.
func ObserveExplainItBackSubmit(outcome string) {
	registerExplainItBackMetrics()
	if outcome == "" {
		outcome = "_unknown"
	}
	explainItBackSubmitsTotal.WithLabelValues(outcome).Inc()
}

// ObserveExplainItBackSchemaFailure increments schema validation failure counter.
func ObserveExplainItBackSchemaFailure() {
	registerExplainItBackMetrics()
	explainItBackSchemaFailuresTotal.Inc()
}

// ObserveExplainItBackRevision increments revision (2nd+ attempt) counter.
func ObserveExplainItBackRevision() {
	registerExplainItBackMetrics()
	explainItBackRevisionsTotal.Inc()
}

func init() {
	RegisterActionHandler(explain_it_back.ID, "submit", handleExplainItBackSubmit)
	RegisterActionHandler(explain_it_back.ID, "instructor_note", handleExplainItBackInstructorNote)
	RegisterActionHandler(explain_it_back.ID, "test_sample", handleExplainItBackTestSample)
}

func handleExplainItBackSubmit(ctx ActionContext) (*ActionResult, error) {
	cfg := explain_it_back.ParseConfig(ctx.ConfigJSON)
	st := explain_it_back.ParseState(ctx.StateJSON)

	var in struct {
		Text           string `json:"text"`
		IdempotencyKey string `json:"idempotencyKey"`
	}
	if len(ctx.Input) > 0 {
		if err := json.Unmarshal(ctx.Input, &in); err != nil {
			return nil, fmt.Errorf("invalid submit input: %w", err)
		}
	}
	text := strings.TrimSpace(in.Text)
	if text == "" {
		text = strings.TrimSpace(st.Draft)
	}
	if text == "" {
		return nil, fmt.Errorf("text is required")
	}
	if len(text) > 8000 {
		return nil, fmt.Errorf("text is too long (max 8000 characters)")
	}
	words := explain_it_back.CountWords(text)
	if words < cfg.MinWords {
		ObserveExplainItBackSubmit("too_short")
		return &ActionResult{
			Result: map[string]any{
				"error":         "too_short",
				"message":       fmt.Sprintf("Write at least %d words (you have %d).", cfg.MinWords, words),
				"wordCount":     words,
				"minWords":      cfg.MinWords,
				"preserveInput": true,
			},
		}, nil
	}
	if words > cfg.MaxWords {
		ObserveExplainItBackSubmit("too_long")
		return &ActionResult{
			Result: map[string]any{
				"error":         "too_long",
				"message":       fmt.Sprintf("Keep it under %d words (you have %d).", cfg.MaxWords, words),
				"wordCount":     words,
				"maxWords":      cfg.MaxWords,
				"preserveInput": true,
			},
		}, nil
	}
	if explain_it_back.AttemptsRemaining(cfg, st) <= 0 {
		ObserveExplainItBackSubmit("max_attempts")
		return &ActionResult{
			Result: map[string]any{
				"error":             "max_attempts",
				"message":           "No attempts remaining for this activity.",
				"attemptsRemaining": 0,
				"preserveInput":     true,
			},
		}, nil
	}

	now := time.Now().UTC()
	if explain_it_back.SubmissionsRemaining(st, cfg.MaxSubmissionsPerDay, now) <= 0 {
		ObserveExplainItBackSubmit("rate_limited")
		resetAt := now.UTC().Truncate(24 * time.Hour).Add(24 * time.Hour)
		return &ActionResult{
			Result: map[string]any{
				"error":         "rate_limited",
				"message":       "You've reached today's submission limit for this activity. Try again after the daily reset.",
				"resetAt":       resetAt.Format(time.RFC3339),
				"maxPerDay":     cfg.MaxSubmissionsPerDay,
				"preserveInput": true,
			},
		}, nil
	}

	screen := ScreenFreeText(text, FilterActionFlag, true)
	if screen.Crisis {
		ObserveExplainItBackSubmit("crisis")
		st.CrisisAcknowledged = true
		st.Draft = text // preserve writing; do not call the model
		patch, err := json.Marshal(st)
		if err != nil {
			return nil, err
		}
		return &ActionResult{
			Result: map[string]any{
				"error":         "filtered",
				"message":       explain_it_back.CrisisSupportMessage,
				"crisis":        true,
				"preserveInput": true,
			},
			StatePatch: patch,
			Status:     explainItBackStatus(ctx.Status, false),
		}, nil
	}
	if screen.Action == FilterActionBlock {
		ObserveExplainItBackSubmit("filtered")
		return &ActionResult{
			Result: map[string]any{
				"error":         "filtered",
				"message":       screen.Guidance,
				"preserveInput": true,
			},
		}, nil
	}

	wasRevision := len(st.Attempts) > 0
	fb, outcome, err := explainItBackResolveFeedback(ctx, cfg, text)
	if err != nil {
		return nil, err
	}
	if wasRevision {
		ObserveExplainItBackRevision()
	}

	attempt := explain_it_back.Attempt{
		At:       now.Format(time.RFC3339),
		Text:     text,
		Feedback: &fb,
	}
	st.Attempts = append(st.Attempts, attempt)
	st.Draft = ""
	explain_it_back.IncrementSubmittedToday(&st, now)
	firstSubstantive := len(st.Attempts) == 1
	if firstSubstantive && st.CompletedAt == "" {
		st.CompletedAt = now.Format(time.RFC3339)
	}

	patch, err := json.Marshal(st)
	if err != nil {
		return nil, err
	}
	ObserveExplainItBackSubmit(outcome)
	result := map[string]any{
		"feedback":          fb,
		"attemptsRemaining": explain_it_back.AttemptsRemaining(cfg, st),
		"wordCount":         words,
		"mode":              string(fb.Mode),
	}
	if cfg.RevealKeyPointsAfterSubmit {
		result["keyPointLabels"] = explain_it_back.KeyPointLabels(cfg)
	}
	status := ""
	if firstSubstantive {
		status = StatusCompleted
	} else {
		status = explainItBackStatus(ctx.Status, false)
	}
	return &ActionResult{
		Result:     result,
		StatePatch: patch,
		Status:     status,
	}, nil
}

func explainItBackResolveFeedback(ctx ActionContext, cfg explain_it_back.Config, text string) (explain_it_back.Feedback, string, error) {
	if !cfg.AIFeedback {
		return explain_it_back.ReviewFeedback(), "review_config", nil
	}
	if ctx.Pool == nil || ctx.Completer == nil {
		// Unit tests / missing deps: degrade to review rather than hard-fail the learner.
		return explain_it_back.ReviewFeedback(), "review_no_deps", nil
	}

	taskPrompt := explain_it_back.BuildTaskPrompt(cfg, ctx.ReadingLevel)
	callOpts := ctctx.CallOpts{
		InstanceID:  ctx.InstanceID,
		CourseID:    ctx.CourseID,
		OrgID:       ctx.OrgID,
		UserID:      ctx.PrincipalID,
		ToolID:      explain_it_back.ID,
		FeatureID:   explain_it_back.FeatureID,
		TaskPrompt:  taskPrompt,
		LearnerText: text,
		Model:       ctx.Model,
		Completer:   ctx.Completer,
		GatewayCfg:  ctx.GatewayCfg,
		BuildOpts: ctctx.BuildOpts{
			Query:         text,
			EnqueueIngest: true,
			TokenBudget:   ctctx.DefaultRequestContextTokens,
		},
		MaxTokens: 400,
	}

	fb, outcome, err := explainItBackCallAndValidate(ctx, cfg, callOpts)
	if err != nil {
		var ge *ctctx.GatewayError
		var be *ctctx.BudgetError
		switch {
		case errors.As(err, &ge):
			return explain_it_back.ReviewFeedback(), "review_gateway", nil
		case errors.As(err, &be):
			return explain_it_back.ReviewFeedback(), "review_budget", nil
		case errors.Is(err, ctctx.ErrProviderUnavailable):
			return explain_it_back.ReviewFeedback(), "review_provider", nil
		default:
			return explain_it_back.ReviewFeedback(), "review_provider", nil
		}
	}
	return fb, outcome, nil
}

func explainItBackCallAndValidate(ctx ActionContext, cfg explain_it_back.Config, opts ctctx.CallOpts) (explain_it_back.Feedback, string, error) {
	call, err := ctctx.RunMediatedCall(ctx.Ctx, ctx.Pool, opts)
	if err != nil {
		return explain_it_back.Feedback{}, "", err
	}
	_ = call.RedactedIn // redaction already applied before egress (AC-6)

	if call.Meta.Provider == aiprovider.ProviderDryRun || explain_it_back.IsDryRunText(call.Text) {
		return explain_it_back.SynthesizeDryRunFeedback(cfg, opts.LearnerText), "ok_ai", nil
	}

	parsed, perr := explain_it_back.ParseModelFeedback(call.Text, cfg)
	if perr == nil {
		return *parsed, "ok_ai", nil
	}
	ObserveExplainItBackSchemaFailure()

	// One retry on malformed structured output (FR-11 / AC-3).
	retry, err := ctctx.RunMediatedCall(ctx.Ctx, ctx.Pool, opts)
	if err != nil {
		return explain_it_back.ReviewFeedback(), "review_schema", nil
	}
	if retry.Meta.Provider == aiprovider.ProviderDryRun || explain_it_back.IsDryRunText(retry.Text) {
		return explain_it_back.SynthesizeDryRunFeedback(cfg, opts.LearnerText), "ok_ai", nil
	}
	parsed2, perr2 := explain_it_back.ParseModelFeedback(retry.Text, cfg)
	if perr2 != nil {
		ObserveExplainItBackSchemaFailure()
		return explain_it_back.ReviewFeedback(), "review_schema", nil
	}
	return *parsed2, "ok_ai", nil
}

func handleExplainItBackInstructorNote(ctx ActionContext) (*ActionResult, error) {
	cfg := explain_it_back.ParseConfig(ctx.ConfigJSON)
	if !cfg.AllowInstructorNote {
		return nil, fmt.Errorf("instructor notes are disabled for this activity")
	}
	if ctx.InteractRole != "" && ctx.InteractRole != "instructor" && ctx.InteractRole != "ta" {
		return nil, fmt.Errorf("only instructors can leave notes")
	}

	var in struct {
		Text         string `json:"text"`
		EnrollmentID string `json:"enrollmentId"`
	}
	if len(ctx.Input) > 0 {
		if err := json.Unmarshal(ctx.Input, &in); err != nil {
			return nil, fmt.Errorf("invalid instructor_note input: %w", err)
		}
	}
	noteText := strings.TrimSpace(in.Text)
	if noteText == "" {
		return nil, fmt.Errorf("text is required")
	}
	if len(noteText) > 2000 {
		return nil, fmt.Errorf("note is too long (max 2000 characters)")
	}
	screen := ScreenFreeText(noteText, FilterActionFlag, false)
	if screen.Action == FilterActionBlock {
		return &ActionResult{
			Result: map[string]any{
				"error":   "filtered",
				"message": screen.Guidance,
			},
		}, nil
	}

	note := explain_it_back.InstructorNote{
		Text: noteText,
		At:   explain_it_back.NowRFC3339(),
		By:   ctx.PrincipalID.String(),
	}

	targetID := ctx.EnrollmentID
	if strings.TrimSpace(in.EnrollmentID) != "" {
		parsed, err := uuid.Parse(strings.TrimSpace(in.EnrollmentID))
		if err != nil {
			return nil, fmt.Errorf("invalid enrollmentId")
		}
		targetID = parsed
	}

	// When targeting another enrollment, write via pool and do not patch the caller's state.
	if targetID != ctx.EnrollmentID {
		if ctx.Pool == nil {
			return nil, fmt.Errorf("instructor note requires database")
		}
		current, err := ctrepo.GetState(ctx.Ctx, ctx.Pool, ctx.InstanceID, targetID)
		if err != nil {
			return nil, err
		}
		st := explain_it_back.EmptyState()
		rev := int64(0)
		status := StatusInProgress
		if current != nil {
			st = explain_it_back.ParseState(current.StateJSON)
			rev = current.Revision
			status = current.Status
		}
		st.InstructorNote = &note
		patch, err := json.Marshal(st)
		if err != nil {
			return nil, err
		}
		if _, err := ctrepo.UpsertStateWithStatus(ctx.Ctx, ctx.Pool, ctx.InstanceID, targetID, ctx.PrincipalID, patch, rev, status, 0); err != nil {
			return nil, fmt.Errorf("failed to save instructor note: %w", err)
		}
		ObserveExplainItBackSubmit("instructor_note")
		return &ActionResult{
			Result: map[string]any{
				"instructorNote": note,
				"enrollmentId":   targetID.String(),
			},
		}, nil
	}

	st := explain_it_back.ParseState(ctx.StateJSON)
	st.InstructorNote = &note
	patch, err := json.Marshal(st)
	if err != nil {
		return nil, err
	}
	ObserveExplainItBackSubmit("instructor_note")
	return &ActionResult{
		Result: map[string]any{
			"instructorNote": note,
		},
		StatePatch: patch,
		Status:     explainItBackStatus(ctx.Status, false),
	}, nil
}

func handleExplainItBackTestSample(ctx ActionContext) (*ActionResult, error) {
	cfg := explain_it_back.ParseConfig(ctx.ConfigJSON)
	var in struct {
		Text string `json:"text"`
	}
	if len(ctx.Input) > 0 {
		if err := json.Unmarshal(ctx.Input, &in); err != nil {
			return nil, fmt.Errorf("invalid test_sample input: %w", err)
		}
	}
	text := strings.TrimSpace(in.Text)
	if text == "" {
		return nil, fmt.Errorf("text is required")
	}
	if !explain_it_back.MeetsLengthGuide(cfg, text) {
		return &ActionResult{
			Result: map[string]any{
				"error":     "length",
				"message":   fmt.Sprintf("Sample must be between %d and %d words.", cfg.MinWords, cfg.MaxWords),
				"wordCount": explain_it_back.CountWords(text),
			},
		}, nil
	}
	fb, outcome, err := explainItBackResolveFeedback(ctx, cfg, text)
	if err != nil {
		return nil, err
	}
	ObserveExplainItBackSubmit("test_sample_" + outcome)
	return &ActionResult{
		Result: map[string]any{
			"feedback":       fb,
			"keyPointLabels": explain_it_back.KeyPointLabels(cfg),
			"mode":           string(fb.Mode),
		},
	}, nil
}

func explainItBackStatus(current string, complete bool) string {
	if complete {
		return StatusCompleted
	}
	switch current {
	case StatusSubmitted, StatusCompleted, StatusInProgress:
		return ""
	default:
		return StatusInProgress
	}
}
