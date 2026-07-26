package contenttools

import (
	"encoding/json"
	"strings"

	"github.com/lextures/lextures/server/internal/service/contenttools/tools/noop_probe"
)

func init() {
	RegisterActionHandler(noop_probe.ID, "grade", handleNoopProbeGrade)
}

type noopProbeConfig struct {
	Prompt      string `json:"prompt"`
	AnswerKey   string `json:"answerKey"`
	MaxAttempts int    `json:"maxAttempts"`
}

type noopProbeState struct {
	Response string `json:"response"`
	Attempts int    `json:"attempts"`
}

func handleNoopProbeGrade(ctx ActionContext) (*ActionResult, error) {
	var cfg noopProbeConfig
	_ = json.Unmarshal(ctx.ConfigJSON, &cfg)
	var st noopProbeState
	_ = json.Unmarshal(ctx.StateJSON, &st)

	if len(ctx.Input) > 0 {
		var in struct {
			Response string `json:"response"`
		}
		if err := json.Unmarshal(ctx.Input, &in); err == nil && strings.TrimSpace(in.Response) != "" {
			st.Response = in.Response
		}
	}
	st.Attempts++
	if cfg.MaxAttempts > 0 && st.Attempts > cfg.MaxAttempts {
		max := 1.0
		raw := 0.0
		patch, _ := json.Marshal(st)
		return &ActionResult{
			Result:     map[string]any{"correct": false, "reason": "max_attempts"},
			StatePatch: patch,
			Status:     StatusCompleted,
			ScoreRaw:   &raw,
			ScoreMax:   &max,
		}, nil
	}

	expected := strings.TrimSpace(strings.ToLower(cfg.AnswerKey))
	got := strings.TrimSpace(strings.ToLower(st.Response))
	correct := expected != "" && expected == got
	max := 1.0
	raw := 0.0
	if correct {
		raw = 1.0
	}
	patch, _ := json.Marshal(st)
	status := StatusSubmitted
	if correct || (cfg.MaxAttempts > 0 && st.Attempts >= cfg.MaxAttempts) {
		status = StatusCompleted
	}
	return &ActionResult{
		Result:     map[string]any{"correct": correct},
		StatePatch: patch,
		Status:     status,
		ScoreRaw:   &raw,
		ScoreMax:   &max,
	}, nil
}
