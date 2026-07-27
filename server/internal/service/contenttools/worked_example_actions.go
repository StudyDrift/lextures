package contenttools

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/lextures/lextures/server/internal/service/contenttools/tools/worked_example"
)

func init() {
	RegisterActionHandler(worked_example.ID, "prepare", handleWorkedExamplePrepare)
	RegisterActionHandler(worked_example.ID, "check_step", handleWorkedExampleCheckStep)
	RegisterActionHandler(worked_example.ID, "hint", handleWorkedExampleHint)
	RegisterActionHandler(worked_example.ID, "reveal_step", handleWorkedExampleRevealStep)
	RegisterActionHandler(worked_example.ID, "reveal_all", handleWorkedExampleRevealAll)
	RegisterActionHandler(worked_example.ID, "verify", handleWorkedExampleVerify)
}

func handleWorkedExamplePrepare(ctx ActionContext) (*ActionResult, error) {
	cfg := worked_example.ParseConfig(ctx.ConfigJSON)
	st := worked_example.ParseState(ctx.StateJSON)
	blanked := workedExampleBlanked(cfg, ctx.EnrollmentID.String())
	ids := make([]string, 0, len(blanked))
	for _, step := range cfg.Steps {
		if blanked[step.ID] {
			ids = append(ids, step.ID)
		}
	}
	st.BlankedStepIDs = ids
	if st.CurrentStepID == "" {
		st.CurrentStepID = worked_example.FirstIncompleteBlanked(cfg, st, blanked)
	}
	patch, err := json.Marshal(st)
	if err != nil {
		return nil, err
	}
	return &ActionResult{
		Result: map[string]any{
			"blankedStepIds": ids,
			"currentStepId":  st.CurrentStepID,
		},
		StatePatch: patch,
		Status:     workedExampleStatus(ctx.Status, false),
	}, nil
}

func workedExampleBlanked(cfg worked_example.Config, enrollmentID string) map[string]bool {
	return worked_example.ResolveBlanked(cfg, worked_example.EnrollmentSeed(enrollmentID))
}

// workedExampleStatus advances status without downgrading (CT.3 transitions are monotonic).
func workedExampleStatus(current string, complete bool) string {
	if complete {
		return StatusCompleted
	}
	switch current {
	case StatusSubmitted, StatusCompleted, StatusInProgress:
		return "" // keep current; empty means no status change in the HTTP layer
	default:
		return StatusInProgress
	}
}

func handleWorkedExampleCheckStep(ctx ActionContext) (*ActionResult, error) {
	cfg := worked_example.ParseConfig(ctx.ConfigJSON)
	st := worked_example.ParseState(ctx.StateJSON)
	blanked := workedExampleBlanked(cfg, ctx.EnrollmentID.String())

	var in struct {
		StepID string `json:"stepId"`
		Value  string `json:"value"`
	}
	if len(ctx.Input) > 0 {
		if err := json.Unmarshal(ctx.Input, &in); err != nil {
			return nil, fmt.Errorf("invalid check_step input: %w", err)
		}
	}
	in.StepID = strings.TrimSpace(in.StepID)
	if in.StepID == "" {
		return nil, fmt.Errorf("stepId is required")
	}
	step := worked_example.FindStep(cfg, in.StepID)
	if step == nil {
		return nil, fmt.Errorf("unknown stepId")
	}
	if !blanked[in.StepID] {
		ObserveWorkedExampleCheck("not_blanked")
		return &ActionResult{
			Result: map[string]any{
				"error":   "not_blanked",
				"message": "This step is not blanked for you.",
			},
		}, nil
	}
	if !worked_example.StepUnlocked(cfg, st, in.StepID, blanked) {
		ObserveWorkedExampleCheck("sequential_locked")
		return &ActionResult{
			Result: map[string]any{
				"error":   "sequential_locked",
				"message": "Complete the previous step first.",
			},
		}, nil
	}
	if worked_example.StepCompleted(st, in.StepID) {
		ObserveWorkedExampleCheck("already_complete")
		return &ActionResult{
			Result: map[string]any{
				"error":   "already_complete",
				"message": "This step is already complete.",
			},
		}, nil
	}
	remaining := worked_example.AttemptsRemaining(cfg, st, in.StepID)
	if remaining == 0 {
		ObserveWorkedExampleCheck("max_attempts")
		return &ActionResult{
			Result: map[string]any{
				"error":             "max_attempts",
				"message":           "No attempts remaining. Use Show me this step to continue.",
				"attemptsRemaining": 0,
				"canReveal":         true,
			},
		}, nil
	}

	value := strings.TrimSpace(in.Value)
	sp := st.Steps[in.StepID]
	if value == "" && sp.Draft != "" {
		value = sp.Draft
	}
	if value == "" {
		return nil, fmt.Errorf("value is required")
	}

	if step.Blank != nil && (step.Blank.Type == worked_example.BlankText || step.Blank.Type == worked_example.BlankExpression) {
		screen := ScreenFreeText(value, FilterActionFlag, true)
		if screen.Action == FilterActionBlock || screen.Crisis {
			ObserveWorkedExampleCheck("filtered")
			return &ActionResult{
				Result: map[string]any{
					"error":         "filtered",
					"message":       screen.Guidance,
					"preserveInput": true,
					"crisis":        screen.Crisis,
				},
			}, nil
		}
	}

	if sp.StartedAt == "" {
		sp.StartedAt = worked_example.NowRFC3339()
	}
	grade := worked_example.GradeStep(cfg, *step, value)
	sp.Attempts = append(sp.Attempts, worked_example.Attempt{
		Value:  value,
		Result: grade.Result,
		At:     worked_example.NowRFC3339(),
	})
	sp.Draft = ""
	if grade.Result == worked_example.ResultCorrect || grade.Result == worked_example.ResultNeedsReview {
		sp.CompletedAt = worked_example.NowRFC3339()
	}
	st.Steps[in.StepID] = sp

	next := ""
	if grade.Result == worked_example.ResultCorrect || grade.Result == worked_example.ResultNeedsReview {
		next = worked_example.NextBlankedStep(cfg, st, in.StepID, blanked)
		st.CurrentStepID = next
		if next == "" {
			st.CurrentStepID = in.StepID
		}
	} else {
		st.CurrentStepID = in.StepID
	}

	raw, max := worked_example.ComputeScore(cfg, st, blanked)
	st.ScoreRaw = &raw
	st.ScoreMax = &max

	complete := worked_example.AllBlankedComplete(cfg, st, blanked)
	if complete {
		st.CompletedAt = worked_example.NowRFC3339()
	}

	attemptsLeft := worked_example.AttemptsRemaining(cfg, st, in.StepID)
	result := map[string]any{
		"result":            string(grade.Result),
		"attemptsRemaining": attemptsLeft,
		"stepId":            in.StepID,
		"canReveal":         attemptsLeft == 0 && grade.Result == worked_example.ResultIncorrect,
	}
	if grade.Feedback != "" {
		result["feedback"] = grade.Feedback
	}
	if next != "" {
		result["nextStep"] = next
	}

	patch, err := json.Marshal(st)
	if err != nil {
		return nil, err
	}
	ObserveWorkedExampleCheck(string(grade.Result))
	if grade.Result == worked_example.ResultNeedsReview {
		ObserveWorkedExampleUndecidable()
	}

	out := &ActionResult{
		Result:     result,
		StatePatch: patch,
		Status:     workedExampleStatus(ctx.Status, complete),
	}
	if !cfg.PracticeOnly {
		out.ScoreRaw = &raw
		out.ScoreMax = &max
	}
	return out, nil
}

func handleWorkedExampleHint(ctx ActionContext) (*ActionResult, error) {
	cfg := worked_example.ParseConfig(ctx.ConfigJSON)
	st := worked_example.ParseState(ctx.StateJSON)
	blanked := workedExampleBlanked(cfg, ctx.EnrollmentID.String())

	var in struct {
		StepID string `json:"stepId"`
	}
	if len(ctx.Input) > 0 {
		if err := json.Unmarshal(ctx.Input, &in); err != nil {
			return nil, fmt.Errorf("invalid hint input: %w", err)
		}
	}
	in.StepID = strings.TrimSpace(in.StepID)
	if in.StepID == "" {
		return nil, fmt.Errorf("stepId is required")
	}
	step := worked_example.FindStep(cfg, in.StepID)
	if step == nil {
		return nil, fmt.Errorf("unknown stepId")
	}
	if !blanked[in.StepID] {
		return &ActionResult{Result: map[string]any{"error": "not_blanked"}}, nil
	}
	if !worked_example.StepUnlocked(cfg, st, in.StepID, blanked) {
		return &ActionResult{Result: map[string]any{"error": "sequential_locked"}}, nil
	}

	sp := st.Steps[in.StepID]
	if sp.StartedAt == "" {
		sp.StartedAt = worked_example.NowRFC3339()
	}
	hints := step.Hints
	if sp.HintsUsed >= len(hints) {
		ObserveWorkedExampleHint("exhausted")
		patch, err := json.Marshal(st)
		if err != nil {
			return nil, err
		}
		return &ActionResult{
			Result: map[string]any{
				"noMoreHints":    true,
				"hintsRemaining": 0,
				"stepId":         in.StepID,
			},
			StatePatch: patch,
		}, nil
	}
	hint := hints[sp.HintsUsed]
	sp.HintsUsed++
	st.Steps[in.StepID] = sp
	patch, err := json.Marshal(st)
	if err != nil {
		return nil, err
	}
	ObserveWorkedExampleHint("ok")
	return &ActionResult{
		Result: map[string]any{
			"hint":           hint,
			"hintsRemaining": len(hints) - sp.HintsUsed,
			"level":          sp.HintsUsed,
			"stepId":         in.StepID,
		},
		StatePatch: patch,
		Status:     workedExampleStatus(ctx.Status, false),
	}, nil
}

func handleWorkedExampleRevealStep(ctx ActionContext) (*ActionResult, error) {
	cfg := worked_example.ParseConfig(ctx.ConfigJSON)
	st := worked_example.ParseState(ctx.StateJSON)
	blanked := workedExampleBlanked(cfg, ctx.EnrollmentID.String())

	var in struct {
		StepID string `json:"stepId"`
	}
	if len(ctx.Input) > 0 {
		if err := json.Unmarshal(ctx.Input, &in); err != nil {
			return nil, fmt.Errorf("invalid reveal_step input: %w", err)
		}
	}
	in.StepID = strings.TrimSpace(in.StepID)
	if in.StepID == "" {
		return nil, fmt.Errorf("stepId is required")
	}
	step := worked_example.FindStep(cfg, in.StepID)
	if step == nil {
		return nil, fmt.Errorf("unknown stepId")
	}
	if !blanked[in.StepID] {
		return &ActionResult{Result: map[string]any{"error": "not_blanked"}}, nil
	}
	if !worked_example.StepUnlocked(cfg, st, in.StepID, blanked) {
		return &ActionResult{Result: map[string]any{"error": "sequential_locked"}}, nil
	}

	used := worked_example.AttemptsUsed(st, in.StepID)
	already := worked_example.StepCompleted(st, in.StepID)
	if !already && used < cfg.AttemptsPerStep {
		return &ActionResult{
			Result: map[string]any{
				"error":   "reveal_not_ready",
				"message": "Use your attempts before revealing this step.",
			},
		}, nil
	}

	sp := st.Steps[in.StepID]
	sp.Revealed = true
	if sp.CompletedAt == "" {
		sp.CompletedAt = worked_example.NowRFC3339()
	}
	st.Steps[in.StepID] = sp

	next := worked_example.NextBlankedStep(cfg, st, in.StepID, blanked)
	st.CurrentStepID = next
	if next == "" {
		st.CurrentStepID = in.StepID
	}

	raw, max := worked_example.ComputeScore(cfg, st, blanked)
	st.ScoreRaw = &raw
	st.ScoreMax = &max

	complete := worked_example.AllBlankedComplete(cfg, st, blanked)
	if complete {
		st.CompletedAt = worked_example.NowRFC3339()
	}

	patch, err := json.Marshal(st)
	if err != nil {
		return nil, err
	}
	ObserveWorkedExampleReveal("step")
	result := map[string]any{
		"stepId":          in.StepID,
		"explanation":     step.Explanation,
		"expectedDisplay": worked_example.ExpectedDisplay(*step),
		"revealed":        true,
	}
	if next != "" {
		result["nextStep"] = next
	}
	out := &ActionResult{
		Result:     result,
		StatePatch: patch,
		Status:     workedExampleStatus(ctx.Status, complete),
	}
	if !cfg.PracticeOnly {
		out.ScoreRaw = &raw
		out.ScoreMax = &max
	}
	return out, nil
}

func handleWorkedExampleRevealAll(ctx ActionContext) (*ActionResult, error) {
	cfg := worked_example.ParseConfig(ctx.ConfigJSON)
	st := worked_example.ParseState(ctx.StateJSON)
	if !cfg.AllowRevealAll {
		return &ActionResult{
			Result: map[string]any{
				"error":   "reveal_forbidden",
				"message": "Full solution reveal is not enabled.",
			},
		}, nil
	}
	blanked := workedExampleBlanked(cfg, ctx.EnrollmentID.String())
	now := worked_example.NowRFC3339()
	stepsOut := make([]map[string]any, 0, len(cfg.Steps))
	for _, step := range cfg.Steps {
		if blanked[step.ID] {
			sp := st.Steps[step.ID]
			sp.Revealed = true
			if sp.CompletedAt == "" {
				sp.CompletedAt = now
			}
			st.Steps[step.ID] = sp
		}
		stepsOut = append(stepsOut, map[string]any{
			"stepId":          step.ID,
			"explanation":     step.Explanation,
			"expectedDisplay": worked_example.ExpectedDisplay(step),
			"text":            step.Text,
		})
	}
	st.RevealAllAt = now
	st.CompletedAt = now
	raw, max := worked_example.ComputeScore(cfg, st, blanked)
	st.ScoreRaw = &raw
	st.ScoreMax = &max
	patch, err := json.Marshal(st)
	if err != nil {
		return nil, err
	}
	ObserveWorkedExampleReveal("all")
	out := &ActionResult{
		Result: map[string]any{
			"steps":    stepsOut,
			"revealed": true,
		},
		StatePatch: patch,
		Status:     StatusCompleted,
	}
	if !cfg.PracticeOnly {
		out.ScoreRaw = &raw
		out.ScoreMax = &max
	}
	return out, nil
}

func handleWorkedExampleVerify(ctx ActionContext) (*ActionResult, error) {
	cfg := worked_example.ParseConfig(ctx.ConfigJSON)
	if len(ctx.Input) > 0 {
		var in struct {
			Config json.RawMessage `json:"config"`
		}
		if err := json.Unmarshal(ctx.Input, &in); err == nil && len(in.Config) > 0 {
			cfg = worked_example.ParseConfig(in.Config)
		}
	}
	results := make([]map[string]any, 0, len(cfg.Steps))
	allOK := true
	for _, step := range cfg.Steps {
		if step.Blank == nil {
			continue
		}
		g := worked_example.VerifyExpected(cfg, step)
		ok := g.Result == worked_example.ResultCorrect
		if !ok {
			allOK = false
		}
		results = append(results, map[string]any{
			"stepId": step.ID,
			"ok":     ok,
			"result": string(g.Result),
		})
	}
	return &ActionResult{
		Result: map[string]any{
			"ok":      allOK,
			"results": results,
		},
	}, nil
}
