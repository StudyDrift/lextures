package contenttools

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/lextures/lextures/server/internal/service/contenttools/tools/parameter_explorer"
)

func init() {
	RegisterActionHandler(parameter_explorer.ID, "checkpoint", handleParameterExplorerCheckpoint)
	RegisterActionHandler(parameter_explorer.ID, "submit_answer", handleParameterExplorerSubmitAnswer)
	RegisterActionHandler(parameter_explorer.ID, "reset_defaults", handleParameterExplorerResetDefaults)
}

func handleParameterExplorerCheckpoint(ctx ActionContext) (*ActionResult, error) {
	cfg := parameter_explorer.ParseConfig(ctx.ConfigJSON)
	st := parameter_explorer.ParseState(ctx.StateJSON)

	if err := parameter_explorer.ValidateConfigForAuthoring(cfg); err != nil {
		ObserveParameterExplorerCheckpoint("invalid_config")
		return &ActionResult{Result: map[string]any{
			"error":   "invalid_config",
			"message": err.Error(),
		}}, nil
	}

	var in struct {
		PromptID string         `json:"promptId"`
		Params   map[string]any `json:"params"`
	}
	if len(ctx.Input) > 0 {
		if err := json.Unmarshal(ctx.Input, &in); err != nil {
			return nil, fmt.Errorf("invalid checkpoint input: %w", err)
		}
	}
	promptID := strings.TrimSpace(in.PromptID)
	if promptID == "" {
		ObserveParameterExplorerCheckpoint("invalid")
		return &ActionResult{Result: map[string]any{"error": "invalid", "message": "promptId required"}}, nil
	}

	var prompt *parameter_explorer.NoticingPrompt
	for i := range cfg.NoticingPrompts {
		if cfg.NoticingPrompts[i].ID == promptID {
			prompt = &cfg.NoticingPrompts[i]
			break
		}
	}
	if prompt == nil {
		ObserveParameterExplorerCheckpoint("unknown_prompt")
		return &ActionResult{Result: map[string]any{"error": "unknown_prompt", "message": "unknown prompt"}}, nil
	}

	params := in.Params
	if len(params) == 0 {
		params = st.Params
	}
	params = parameter_explorer.ClampParams(cfg, params)
	st.Params = params
	st = parameter_explorer.AppendTrace(st, params, "")

	// Already unlocked — idempotent success.
	if hit, ok := st.Checkpoints[promptID]; ok && hit != "" {
		ObserveParameterExplorerCheckpoint("already")
		return &ActionResult{
			Result: map[string]any{
				"ok":        true,
				"unlocked":  true,
				"hitAt":     hit,
				"already":   true,
				"promptId":  promptID,
			},
			StatePatch: mustJSON(st),
		}, nil
	}

	ok, err := parameter_explorer.EvalUnlock(prompt.UnlockWhen, params)
	if err != nil {
		ObserveParameterExplorerCheckpoint("bad_predicate")
		return &ActionResult{Result: map[string]any{"error": "bad_predicate", "message": err.Error()}}, nil
	}
	if !ok {
		ObserveParameterExplorerCheckpoint("not_met")
		return &ActionResult{
			Result: map[string]any{
				"ok":       false,
				"unlocked": false,
				"promptId": promptID,
				"message":  "checkpoint not met",
			},
			StatePatch: mustJSON(st),
		}, nil
	}

	hitAt := parameter_explorer.NowRFC3339()
	st.Checkpoints[promptID] = hitAt
	bins := parameter_explorer.ParamBins(cfg, params)
	ObserveParameterExplorerCheckpoint("hit")
	return &ActionResult{
		Result: map[string]any{
			"ok":       true,
			"unlocked": true,
			"hitAt":    hitAt,
			"promptId": promptID,
			"paramBins": bins,
			"checkpointCount": len(parameter_explorer.CheckpointPrompts(cfg)),
		},
		StatePatch: mustJSON(st),
	}, nil
}

func handleParameterExplorerSubmitAnswer(ctx ActionContext) (*ActionResult, error) {
	cfg := parameter_explorer.ParseConfig(ctx.ConfigJSON)
	st := parameter_explorer.ParseState(ctx.StateJSON)

	var in struct {
		PromptID string         `json:"promptId"`
		Answer   string         `json:"answer"`
		Params   map[string]any `json:"params"`
	}
	if len(ctx.Input) > 0 {
		if err := json.Unmarshal(ctx.Input, &in); err != nil {
			return nil, fmt.Errorf("invalid submit_answer input: %w", err)
		}
	}
	promptID := strings.TrimSpace(in.PromptID)
	answer := strings.TrimSpace(in.Answer)
	if promptID == "" {
		ObserveParameterExplorerAnswer("invalid")
		return &ActionResult{Result: map[string]any{"error": "invalid", "message": "promptId required"}}, nil
	}

	var prompt *parameter_explorer.NoticingPrompt
	for i := range cfg.NoticingPrompts {
		if cfg.NoticingPrompts[i].ID == promptID {
			prompt = &cfg.NoticingPrompts[i]
			break
		}
	}
	if prompt == nil {
		ObserveParameterExplorerAnswer("unknown_prompt")
		return &ActionResult{Result: map[string]any{"error": "unknown_prompt"}}, nil
	}

	// Gate on checkpoint when configured.
	if strings.TrimSpace(prompt.UnlockWhen) != "" {
		if _, hit := st.Checkpoints[promptID]; !hit {
			// Allow server re-check with submitted params.
			params := in.Params
			if len(params) == 0 {
				params = st.Params
			}
			params = parameter_explorer.ClampParams(cfg, params)
			ok, err := parameter_explorer.EvalUnlock(prompt.UnlockWhen, params)
			if err != nil || !ok {
				ObserveParameterExplorerAnswer("locked")
				return &ActionResult{Result: map[string]any{
					"error":   "locked",
					"message": "Reach the checkpoint before answering.",
				}}, nil
			}
			st.Checkpoints[promptID] = parameter_explorer.NowRFC3339()
			st.Params = params
		}
	}

	if prompt.Kind == "text" && answer != "" {
		screen := ScreenFreeText(answer, FilterActionFlag, true)
		if screen.Action == FilterActionBlock {
			ObserveParameterExplorerAnswer("filtered")
			return &ActionResult{Result: map[string]any{
				"error":         "filtered",
				"message":       screen.Guidance,
				"preserveInput": true,
			}}, nil
		}
		if screen.Crisis {
			ObserveParameterExplorerAnswer("crisis")
			return &ActionResult{Result: map[string]any{
				"error":         "filtered",
				"message":       screen.Guidance,
				"crisis":        true,
				"preserveInput": true,
			}}, nil
		}
	}

	if prompt.Kind == "choice" {
		valid := false
		for _, opt := range prompt.Options {
			if opt.Value == answer {
				valid = true
				break
			}
		}
		if !valid && answer != "" {
			ObserveParameterExplorerAnswer("invalid_choice")
			return &ActionResult{Result: map[string]any{"error": "invalid_choice"}}, nil
		}
	}

	if len(in.Params) > 0 {
		st.Params = parameter_explorer.ClampParams(cfg, in.Params)
		st = parameter_explorer.AppendTrace(st, st.Params, "")
	}

	st.Answers[promptID] = answer
	status := StatusInProgress
	if parameter_explorer.IsComplete(cfg, st) {
		st.CompletedAt = parameter_explorer.NowRFC3339()
		status = StatusCompleted
		ObserveParameterExplorerAnswer("completed")
	} else {
		ObserveParameterExplorerAnswer("ok")
	}

	return &ActionResult{
		Result: map[string]any{
			"ok":        true,
			"promptId":  promptID,
			"completed": status == StatusCompleted,
		},
		StatePatch: mustJSON(st),
		Status:     status,
	}, nil
}

func handleParameterExplorerResetDefaults(ctx ActionContext) (*ActionResult, error) {
	cfg := parameter_explorer.ParseConfig(ctx.ConfigJSON)
	st := parameter_explorer.EmptyState()
	st.Params = parameter_explorer.DefaultParams(cfg)
	ObserveParameterExplorerReset()
	return &ActionResult{
		Result:     map[string]any{"ok": true},
		StatePatch: mustJSON(st),
		// Status stays as-is: actions are forward-only; CT.4 self-reset clears status.
	}, nil
}

func mustJSON(v any) json.RawMessage {
	b, err := json.Marshal(v)
	if err != nil {
		return json.RawMessage(`{}`)
	}
	return b
}

var (
	parameterExplorerMetricsOnce sync.Once

	parameterExplorerCheckpointsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "lextures",
		Name:      "content_tool_explorer_checkpoints_total",
		Help:      "Parameter Explorer checkpoint outcomes (CT.16).",
	}, []string{"outcome"})

	parameterExplorerAnswersTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "lextures",
		Name:      "content_tool_explorer_answers_total",
		Help:      "Parameter Explorer answer submit outcomes (CT.16).",
	}, []string{"outcome"})

	parameterExplorerResetsTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: "lextures",
		Name:      "content_tool_explorer_resets_total",
		Help:      "Parameter Explorer in-tool reset_defaults actions (CT.16).",
	})
)

func registerParameterExplorerMetrics() {
	parameterExplorerMetricsOnce.Do(func() {
		prometheus.MustRegister(
			parameterExplorerCheckpointsTotal,
			parameterExplorerAnswersTotal,
			parameterExplorerResetsTotal,
		)
		parameterExplorerCheckpointsTotal.WithLabelValues("_reserved").Add(0)
		parameterExplorerAnswersTotal.WithLabelValues("_reserved").Add(0)
	})
}

// ObserveParameterExplorerCheckpoint increments checkpoint outcomes (CT.16).
func ObserveParameterExplorerCheckpoint(outcome string) {
	registerParameterExplorerMetrics()
	if outcome == "" {
		outcome = "_unknown"
	}
	parameterExplorerCheckpointsTotal.WithLabelValues(outcome).Inc()
}

// ObserveParameterExplorerAnswer increments answer submit outcomes (CT.16).
func ObserveParameterExplorerAnswer(outcome string) {
	registerParameterExplorerMetrics()
	if outcome == "" {
		outcome = "_unknown"
	}
	parameterExplorerAnswersTotal.WithLabelValues(outcome).Inc()
}

// ObserveParameterExplorerReset increments in-tool reset_defaults (CT.16).
func ObserveParameterExplorerReset() {
	registerParameterExplorerMetrics()
	parameterExplorerResetsTotal.Inc()
}
