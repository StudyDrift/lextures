package contenttools

import (
	"encoding/json"
	"testing"
)

func TestCanTransitionStateStatus(t *testing.T) {
	cases := []struct {
		from, to string
		ok       bool
	}{
		{"not_started", "in_progress", true},
		{"in_progress", "submitted", true},
		{"submitted", "completed", true},
		{"not_started", "completed", true},
		{"submitted", "in_progress", false},
		{"completed", "not_started", false},
		{"in_progress", "in_progress", true},
		{"in_progress", "", true},
		{"bogus", "in_progress", false},
	}
	for _, tc := range cases {
		if got := CanTransitionStateStatus(tc.from, tc.to); got != tc.ok {
			t.Fatalf("%s→%s: got %v want %v", tc.from, tc.to, got, tc.ok)
		}
	}
}

func TestNextStatusOnSave(t *testing.T) {
	got, err := NextStatusOnSave("not_started", "")
	if err != nil || got != "in_progress" {
		t.Fatalf("auto advance: got %q err=%v", got, err)
	}
	got, err = NextStatusOnSave("submitted", "in_progress")
	if err == nil {
		t.Fatalf("expected error for backward transition, got %q", got)
	}
	got, err = NextStatusOnSave("in_progress", "submitted")
	if err != nil || got != "submitted" {
		t.Fatalf("submit: got %q err=%v", got, err)
	}
}

func TestRuntimeReadonlyEnv(t *testing.T) {
	t.Setenv(EnvRuntimeReadonly, "on")
	if !RuntimeReadonly() {
		t.Fatal("expected readonly")
	}
	t.Setenv(EnvRuntimeReadonly, "")
	if RuntimeReadonly() {
		t.Fatal("expected not readonly")
	}
}

func TestNoopProbeGradeAction(t *testing.T) {
	cfg, _ := json.Marshal(map[string]any{"prompt": "Q?", "answerKey": "yes", "maxAttempts": 3})
	st, _ := json.Marshal(map[string]any{"response": "yes", "attempts": 0})
	res, err := DispatchAction(MustDefault().Get("noop_probe"), "grade", ActionContext{
		ConfigJSON: cfg,
		StateJSON:  st,
		Input:      json.RawMessage(`{}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Result["correct"] != true {
		t.Fatalf("expected correct, got %#v", res.Result)
	}
	if res.ScoreRaw == nil || *res.ScoreRaw != 1 {
		t.Fatalf("expected score 1, got %#v", res.ScoreRaw)
	}
	if res.Status != StatusCompleted {
		t.Fatalf("status=%s", res.Status)
	}
}

func TestManifestActionsValidated(t *testing.T) {
	m := MustDefault().Get("noop_probe")
	if m == nil {
		t.Fatal("missing noop_probe")
	}
	if FindAction(m, "grade") == nil {
		t.Fatal("grade action missing")
	}
	if EffectiveConflictPolicy(m) != ConflictServerWins {
		t.Fatalf("conflict=%s", EffectiveConflictPolicy(m))
	}
	if EffectiveAutosaveMs(m) != 1500 {
		t.Fatalf("autosave=%d", EffectiveAutosaveMs(m))
	}
}

func TestMergeStateJSON(t *testing.T) {
	out, err := MergeStateJSON(
		json.RawMessage(`{"a":1,"b":2}`),
		json.RawMessage(`{"b":3,"c":4}`),
	)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(out, &m); err != nil {
		t.Fatal(err)
	}
	if m["a"].(float64) != 1 || m["b"].(float64) != 3 || m["c"].(float64) != 4 {
		t.Fatalf("merge failed: %#v", m)
	}
}
