package contenttools

import (
	"encoding/json"
	"strings"

	"github.com/lextures/lextures/server/internal/service/contenttools/tools/sandbox_probe"
)

func init() {
	RegisterActionHandler(sandbox_probe.ID, "grade", handleSandboxProbeGrade)
}

func handleSandboxProbeGrade(ctx ActionContext) (*ActionResult, error) {
	var cfg struct {
		AnswerKey string `json:"answerKey"`
	}
	_ = json.Unmarshal(ctx.ConfigJSON, &cfg)
	var st struct {
		Response string `json:"response"`
	}
	_ = json.Unmarshal(ctx.StateJSON, &st)
	if len(ctx.Input) > 0 {
		var in struct {
			Response string `json:"response"`
		}
		if err := json.Unmarshal(ctx.Input, &in); err == nil && strings.TrimSpace(in.Response) != "" {
			st.Response = in.Response
		}
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
	if correct {
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
