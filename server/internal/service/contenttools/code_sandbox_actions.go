package contenttools

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/lextures/lextures/server/internal/service/codeexecution"
	"github.com/lextures/lextures/server/internal/service/contenttools/tools/code_sandbox"
)

// codeRunner is overridable in tests.
var codeRunner = func() codeexecution.Service {
	return codeexecution.New()
}

func init() {
	RegisterActionHandler(code_sandbox.ID, "run", handleCodeSandboxRun)
	RegisterActionHandler(code_sandbox.ID, "check", handleCodeSandboxCheck)
	RegisterActionHandler(code_sandbox.ID, "reset_code", handleCodeSandboxResetCode)
	RegisterActionHandler(code_sandbox.ID, "try_reference", handleCodeSandboxTryReference)
}

func codeExecutionAllowed(ctx ActionContext) bool {
	if ctx.CodeExecutionEnabled == nil {
		return true
	}
	return *ctx.CodeExecutionEnabled
}

func codeSandboxUnavailableResult(message string) *ActionResult {
	ObserveCodeSandboxRun("_", "unavailable", "runner_unavailable")
	return &ActionResult{
		Result: map[string]any{
			"error":   "runner_unavailable",
			"message": message,
		},
	}
}

func handleCodeSandboxRun(ctx ActionContext) (*ActionResult, error) {
	if !codeExecutionAllowed(ctx) {
		return codeSandboxUnavailableResult("Code execution is not enabled on this platform."), nil
	}
	cfg := code_sandbox.ParseConfig(ctx.ConfigJSON)
	st := code_sandbox.ParseState(ctx.StateJSON)
	now := time.Now().UTC()

	if !code_sandbox.SupportedLanguage(cfg.Language) {
		return &ActionResult{
			Result: map[string]any{
				"error":   "unsupported_language",
				"message": fmt.Sprintf("Language %q is not supported.", cfg.Language),
			},
		}, nil
	}

	var in struct {
		Code           string `json:"code"`
		Stdin          string `json:"stdin"`
		IdempotencyKey string `json:"idempotencyKey"`
	}
	if len(ctx.Input) > 0 {
		if err := json.Unmarshal(ctx.Input, &in); err != nil {
			return nil, fmt.Errorf("invalid run input: %w", err)
		}
	}
	code := in.Code
	if strings.TrimSpace(code) == "" {
		code = code_sandbox.InitialCode(cfg, st)
	}
	stdin := in.Stdin
	if stdin == "" {
		stdin = cfg.SampleInput
	}

	if rl := code_sandbox.CheckRateLimit(cfg, st, code_sandbox.ActionRun, now); rl != nil {
		ObserveCodeSandboxRun(cfg.Language, "run", "rate_limited")
		return &ActionResult{
			Result: map[string]any{
				"error":   "rate_limited",
				"message": rl.Message,
				"resetAt": rl.ResetAt,
				"limit":   rl.Limit,
				"action":  "run",
			},
		}, nil
	}

	composed := code_sandbox.ComposeCode(cfg, code)
	resp, err := codeRunner().RunTests(ctx.Ctx, codeexecution.RunRequest{
		Runtime: cfg.Language,
		Code:    composed,
		Tests:   code_sandbox.SampleRunTest(stdin),
	})
	if err != nil {
		ObserveCodeSandboxRun(cfg.Language, "run", "error")
		return codeSandboxUnavailableResult("The code runner is temporarily unavailable. Your code was preserved."), nil
	}

	primary := codeexecution.TestResult{}
	if len(resp.Results) > 0 {
		primary = resp.Results[0]
	}
	status := code_sandbox.MapRunnerStatus(resp, primary)
	stdout := code_sandbox.TruncateOutput(primary.ActualOutput, cfg.OutputLimitBytes)
	stderr := code_sandbox.TruncateOutput(primary.Stderr, cfg.OutputLimitBytes)
	if resp.CompileError != "" && stderr == "" {
		stderr = code_sandbox.TruncateOutput(resp.CompileError, cfg.OutputLimitBytes)
	}
	hint := code_sandbox.MatchErrorHint(cfg, stderr)

	at := code_sandbox.NowRFC3339()
	st.Code = code
	st = code_sandbox.RecordRateUsage(st, code_sandbox.ActionRun, now)
	st = code_sandbox.AppendRun(st, code_sandbox.RunRecord{
		At:     at,
		Action: code_sandbox.ActionRun,
		Status: status,
		Stdout: stdout,
		Stderr: stderr,
	}, cfg.MaxRunHistory)

	patch, err := json.Marshal(st)
	if err != nil {
		return nil, err
	}
	ObserveCodeSandboxRun(cfg.Language, "run", string(status))
	result := map[string]any{
		"status": status,
		"stdout": stdout,
		"stderr": stderr,
	}
	if hint != "" {
		result["hint"] = hint
	}
	if strings.Contains(stdout, code_sandbox.TruncationMarker) || strings.Contains(stderr, code_sandbox.TruncationMarker) {
		result["truncated"] = true
	}
	return &ActionResult{Result: result, StatePatch: patch}, nil
}

func handleCodeSandboxCheck(ctx ActionContext) (*ActionResult, error) {
	if !codeExecutionAllowed(ctx) {
		return codeSandboxUnavailableResult("Code execution is not enabled on this platform."), nil
	}
	cfg := code_sandbox.ParseConfig(ctx.ConfigJSON)
	st := code_sandbox.ParseState(ctx.StateJSON)
	now := time.Now().UTC()

	if len(cfg.Tests) == 0 {
		return &ActionResult{
			Result: map[string]any{
				"error":   "no_tests",
				"message": "This sandbox has no tests to check.",
			},
		}, nil
	}
	if !code_sandbox.SupportedLanguage(cfg.Language) {
		return &ActionResult{
			Result: map[string]any{
				"error":   "unsupported_language",
				"message": fmt.Sprintf("Language %q is not supported.", cfg.Language),
			},
		}, nil
	}

	var in struct {
		Code string `json:"code"`
	}
	if len(ctx.Input) > 0 {
		if err := json.Unmarshal(ctx.Input, &in); err != nil {
			return nil, fmt.Errorf("invalid check input: %w", err)
		}
	}
	code := in.Code
	if strings.TrimSpace(code) == "" {
		code = code_sandbox.InitialCode(cfg, st)
	}

	if rl := code_sandbox.CheckRateLimit(cfg, st, code_sandbox.ActionCheck, now); rl != nil {
		ObserveCodeSandboxRun(cfg.Language, "check", "rate_limited")
		return &ActionResult{
			Result: map[string]any{
				"error":   "rate_limited",
				"message": rl.Message,
				"resetAt": rl.ResetAt,
				"limit":   rl.Limit,
				"action":  "check",
			},
		}, nil
	}

	composed := code_sandbox.ComposeCode(cfg, code)
	resp, err := codeRunner().RunTests(ctx.Ctx, codeexecution.RunRequest{
		Runtime: cfg.Language,
		Code:    composed,
		Tests:   code_sandbox.ToRunnerTests(cfg),
	})
	if err != nil {
		ObserveCodeSandboxRun(cfg.Language, "check", "error")
		return codeSandboxUnavailableResult("The code runner is temporarily unavailable. Your code was preserved."), nil
	}

	tests, passed, total, status, stdout, stderr := code_sandbox.GradeCheck(cfg, resp)
	hint := code_sandbox.MatchErrorHint(cfg, stderr)
	at := code_sandbox.NowRFC3339()

	outcomes := make([]code_sandbox.TestOutcome, 0, len(tests))
	for _, tr := range tests {
		outcomes = append(outcomes, code_sandbox.TestOutcome{ID: tr.ID, Passed: tr.Passed})
	}

	st.Code = code
	st = code_sandbox.RecordRateUsage(st, code_sandbox.ActionCheck, now)
	st = code_sandbox.AppendRun(st, code_sandbox.RunRecord{
		At:     at,
		Action: code_sandbox.ActionCheck,
		Status: status,
		Stdout: stdout,
		Stderr: stderr,
		Tests:  outcomes,
	}, cfg.MaxRunHistory)
	st = code_sandbox.UpdateBest(st, passed, total, at)

	lifecycle := ""
	if passed == total && total > 0 {
		st.CompletedAt = at
		lifecycle = StatusCompleted
	} else if st.CompletedAt == "" {
		lifecycle = StatusSubmitted
	}

	patch, err := json.Marshal(st)
	if err != nil {
		return nil, err
	}

	testOut := make([]map[string]any, 0, len(tests))
	for _, tr := range tests {
		entry := map[string]any{
			"id":     tr.ID,
			"name":   tr.Name,
			"passed": tr.Passed,
			"hidden": tr.Hidden,
		}
		if tr.Feedback != "" {
			entry["feedback"] = tr.Feedback
		}
		// Never include input/expectedOutput — even for visible tests in check payload
		// beyond name/pass/feedback (FR-3 for hidden; visible stay free of expected answers).
		testOut = append(testOut, entry)
	}

	result := map[string]any{
		"tests":   testOut,
		"passed":  passed,
		"total":   total,
		"status":  status,
		"stdout":  stdout,
		"stderr":  stderr,
	}
	if hint != "" {
		result["hint"] = hint
	}

	ObserveCodeSandboxRun(cfg.Language, "check", string(status))
	ObserveCodeSandboxTests(cfg.Language, passed, total)

	out := &ActionResult{
		Result:     result,
		StatePatch: patch,
		Status:     lifecycle,
	}
	if code_sandbox.EffectiveScoringMode(cfg) == code_sandbox.ScoringAuto {
		raw := float64(passed)
		max := float64(total)
		out.ScoreRaw = &raw
		out.ScoreMax = &max
	}
	return out, nil
}

func handleCodeSandboxResetCode(ctx ActionContext) (*ActionResult, error) {
	cfg := code_sandbox.ParseConfig(ctx.ConfigJSON)
	st := code_sandbox.ParseState(ctx.StateJSON)
	st.Code = cfg.StarterCode
	// Preserve runs, best, rate, completedAt (FR-11 / AC-6).
	patch, err := json.Marshal(st)
	if err != nil {
		return nil, err
	}
	ObserveCodeSandboxRun(cfg.Language, "reset_code", "ok")
	return &ActionResult{
		Result: map[string]any{
			"ok":   true,
			"code": st.Code,
		},
		StatePatch: patch,
	}, nil
}

func handleCodeSandboxTryReference(ctx ActionContext) (*ActionResult, error) {
	if !codeExecutionAllowed(ctx) {
		return codeSandboxUnavailableResult("Code execution is not enabled on this platform."), nil
	}
	cfg := code_sandbox.ParseConfig(ctx.ConfigJSON)
	var in struct {
		Code  string          `json:"code"`
		Tests json.RawMessage `json:"tests"`
	}
	if err := json.Unmarshal(ctx.Input, &in); err != nil {
		return nil, fmt.Errorf("invalid try_reference input: %w", err)
	}
	if len(in.Tests) > 0 {
		var tests []code_sandbox.TestCase
		if err := json.Unmarshal(in.Tests, &tests); err != nil {
			return nil, fmt.Errorf("invalid try_reference tests: %w", err)
		}
		if len(tests) > code_sandbox.MaxTests {
			tests = tests[:code_sandbox.MaxTests]
		}
		cfg.Tests = tests
	}
	if len(cfg.Tests) == 0 {
		return &ActionResult{
			Result: map[string]any{
				"error":   "no_tests",
				"message": "Add tests before trying a reference solution.",
			},
		}, nil
	}
	if strings.TrimSpace(in.Code) == "" {
		return &ActionResult{
			Result: map[string]any{
				"error":   "empty_code",
				"message": "Reference solution code is required.",
			},
		}, nil
	}
	composed := code_sandbox.ComposeCode(cfg, in.Code)
	resp, err := codeRunner().RunTests(ctx.Ctx, codeexecution.RunRequest{
		Runtime: cfg.Language,
		Code:    composed,
		Tests:   code_sandbox.ToRunnerTests(cfg),
	})
	if err != nil {
		return codeSandboxUnavailableResult("The code runner is temporarily unavailable."), nil
	}
	tests, passed, total, status, _, stderr := code_sandbox.GradeCheck(cfg, resp)
	testOut := make([]map[string]any, 0, len(tests))
	for _, tr := range tests {
		testOut = append(testOut, map[string]any{
			"id":     tr.ID,
			"name":   tr.Name,
			"passed": tr.Passed,
			"hidden": tr.Hidden,
		})
	}
	ObserveCodeSandboxRun(cfg.Language, "try_reference", string(status))
	return &ActionResult{
		Result: map[string]any{
			"tests":  testOut,
			"passed": passed,
			"total":  total,
			"status": status,
			"stderr": stderr,
			"ok":     passed == total && total > 0,
		},
	}, nil
}
