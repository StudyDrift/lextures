package contenttools

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/lextures/lextures/server/internal/service/codeexecution"
	"github.com/lextures/lextures/server/internal/service/contenttools/tools/code_sandbox"
)

func sampleSandboxConfig(t *testing.T, overrides map[string]any) json.RawMessage {
	t.Helper()
	base := map[string]any{
		"language":    "python",
		"prompt":      "Write a function that doubles n",
		"starterCode": "n = int(input())\nprint(n * 2)\n",
		"sampleInput": "3",
		"tests": []map[string]any{
			{
				"id":             "t1",
				"name":           "doubles 3",
				"input":          "3",
				"expectedOutput": "6",
				"hidden":         false,
				"feedback":       "Remember to print the result.",
			},
			{
				"id":             "t2",
				"name":           "doubles 10",
				"input":          "10",
				"expectedOutput": "20",
				"hidden":         true,
				"feedback":       "Hidden case failed.",
			},
		},
		"runLimitPerHour":   30,
		"checkLimitPerHour": 20,
		"scoringMode":       "auto",
	}
	for k, v := range overrides {
		base[k] = v
	}
	raw, err := json.Marshal(base)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func TestCodeSandboxRunAndCheck(t *testing.T) {
	if _, err := codeexecution.New().Health(context.Background()); err != nil {
		t.Skip("codeexecution runtime unavailable:", err)
	}
	cfg := sampleSandboxConfig(t, nil)
	ctx := ActionContext{
		Ctx:        context.Background(),
		ConfigJSON: cfg,
		StateJSON:  json.RawMessage(`{}`),
		Input:      json.RawMessage(`{"code":"n = int(input())\\nprint(n * 2)\\n"}`),
	}
	// Fix escaped newlines in JSON properly.
	ctx.Input = json.RawMessage(`{"code":"n = int(input())\nprint(n * 2)\n","stdin":"3"}`)

	runRes, err := handleCodeSandboxRun(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if runRes.Result["error"] != nil {
		t.Fatalf("run error: %v", runRes.Result)
	}
	if runRes.Result["status"] != code_sandbox.StatusOK && runRes.Result["status"] != code_sandbox.StatusRuntimeError {
		// OK or runtime depending on python availability nuances
		if s, _ := runRes.Result["stdout"].(string); !strings.Contains(s, "6") {
			t.Fatalf("unexpected run result: %#v", runRes.Result)
		}
	}
	if s, _ := runRes.Result["stdout"].(string); !strings.Contains(s, "6") {
		t.Fatalf("stdout=%v", runRes.Result["stdout"])
	}

	ctx.StateJSON = runRes.StatePatch
	ctx.Input = json.RawMessage(`{"code":"n = int(input())\nprint(n * 2)\n"}`)
	checkRes, err := handleCodeSandboxCheck(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if checkRes.Result["error"] != nil {
		t.Fatalf("check error: %v", checkRes.Result)
	}
	passed, _ := checkRes.Result["passed"].(int)
	if passed == 0 {
		// json numbers are float64
		if pf, ok := checkRes.Result["passed"].(float64); ok {
			passed = int(pf)
		}
	}
	if passed != 2 {
		t.Fatalf("passed=%v result=%#v", checkRes.Result["passed"], checkRes.Result)
	}
	tests, ok := checkRes.Result["tests"].([]map[string]any)
	if !ok {
		// encoding may yield []any
		rawTests, _ := checkRes.Result["tests"].([]any)
		for _, item := range rawTests {
			m, _ := item.(map[string]any)
			if m["hidden"] == true {
				if _, has := m["input"]; has {
					t.Fatal("hidden test leaked input")
				}
				if _, has := m["expectedOutput"]; has {
					t.Fatal("hidden test leaked expectedOutput")
				}
			}
			for k := range m {
				if k == "input" || k == "expectedOutput" {
					t.Fatalf("test payload leaked %s", k)
				}
			}
		}
	} else {
		_ = tests
	}
	if checkRes.ScoreRaw == nil || *checkRes.ScoreRaw != 2 {
		t.Fatalf("scoreRaw=%v", checkRes.ScoreRaw)
	}
}

func TestCodeSandboxHiddenRedactionInConfig(t *testing.T) {
	reg, err := BuildBuiltinRegistry()
	if err != nil {
		t.Fatal(err)
	}
	m := reg.Get(code_sandbox.ID)
	if m == nil {
		t.Fatal("code_sandbox not registered")
	}
	cfg := sampleSandboxConfig(t, nil)
	redacted, err := RedactSensitiveConfig(m.ConfigSchema, cfg)
	if err != nil {
		t.Fatal(err)
	}
	var doc map[string]any
	if err := json.Unmarshal(redacted, &doc); err != nil {
		t.Fatal(err)
	}
	tests, _ := doc["tests"].([]any)
	if len(tests) != 2 {
		t.Fatalf("tests len=%d", len(tests))
	}
	for _, item := range tests {
		tm := item.(map[string]any)
		if _, ok := tm["input"]; ok {
			t.Fatal("input should be redacted")
		}
		if _, ok := tm["expectedOutput"]; ok {
			t.Fatal("expectedOutput should be redacted")
		}
		if _, ok := tm["feedback"]; ok {
			t.Fatal("feedback should be redacted")
		}
		if tm["name"] == nil || tm["id"] == nil {
			t.Fatal("id/name should remain")
		}
	}
}

func TestCodeSandboxRateLimit(t *testing.T) {
	cfg := sampleSandboxConfig(t, map[string]any{"runLimitPerHour": 1})
	st := code_sandbox.EmptyState()
	now := time.Now().UTC()
	st = code_sandbox.RecordRateUsage(st, code_sandbox.ActionRun, now)
	raw, _ := json.Marshal(st)
	ctx := ActionContext{
		Ctx:        context.Background(),
		ConfigJSON: cfg,
		StateJSON:  raw,
		Input:      json.RawMessage(`{"code":"print(1)"}`),
	}
	res, err := handleCodeSandboxRun(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if res.Result["error"] != "rate_limited" {
		t.Fatalf("expected rate_limited, got %#v", res.Result)
	}
	if res.Result["resetAt"] == nil {
		t.Fatal("expected resetAt")
	}
}

func TestCodeSandboxResetCodeKeepsHistory(t *testing.T) {
	cfg := sampleSandboxConfig(t, nil)
	st := code_sandbox.EmptyState()
	st.Code = "print('edited')"
	st.Runs = []code_sandbox.RunRecord{{
		At: code_sandbox.NowRFC3339(), Action: code_sandbox.ActionRun, Status: code_sandbox.StatusOK,
	}}
	raw, _ := json.Marshal(st)
	res, err := handleCodeSandboxResetCode(ActionContext{
		Ctx: context.Background(), ConfigJSON: cfg, StateJSON: raw, Input: json.RawMessage(`{}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	var next code_sandbox.State
	if err := json.Unmarshal(res.StatePatch, &next); err != nil {
		t.Fatal(err)
	}
	if next.Code != code_sandbox.ParseConfig(cfg).StarterCode {
		t.Fatalf("code=%q", next.Code)
	}
	if len(next.Runs) != 1 {
		t.Fatalf("runs cleared: %d", len(next.Runs))
	}
}

func TestCodeSandboxMaxStateBytesAllowed(t *testing.T) {
	reg, err := BuildBuiltinRegistry()
	if err != nil {
		t.Fatal(err)
	}
	m := reg.Get(code_sandbox.ID)
	if m == nil {
		t.Fatal("missing tool")
	}
	if m.Storage.MaxStateBytes != 128000 {
		t.Fatalf("maxStateBytes=%d", m.Storage.MaxStateBytes)
	}
	// 70KB state must be accepted now that CT.17 raises the cap.
	bigCode := strings.Repeat("x", 70*1024)
	st := map[string]any{"v": 1, "code": bigCode, "runs": []any{}}
	raw, _ := json.Marshal(st)
	if err := ValidateStateJSON(m, raw); err != nil {
		t.Fatalf("expected 70KB state ok, got %v", err)
	}
}
