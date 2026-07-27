package contenttools

import (
	"encoding/json"
	"testing"

	"github.com/lextures/lextures/server/internal/service/contenttools/tools/parameter_explorer"
)

func TestParameterExplorerCheckpoint_HitAndIdempotent(t *testing.T) {
	cfg := parameter_explorer.DefaultConfig()
	cfg.NoticingPrompts = []parameter_explorer.NoticingPrompt{
		{ID: "n1", Text: "Notice steepness", Kind: "text", Required: true, UnlockWhen: "a > 1.5"},
	}
	cfgJSON, _ := json.Marshal(cfg)
	st := parameter_explorer.EmptyState()
	st.Params = map[string]any{"a": 2.0, "b": 0.0, "c": 0.0}
	stJSON, _ := json.Marshal(st)

	res, err := handleParameterExplorerCheckpoint(ActionContext{
		ConfigJSON: cfgJSON,
		StateJSON:  stJSON,
		Input:      mustJSON(map[string]any{"promptId": "n1", "params": st.Params}),
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Result["unlocked"] != true {
		t.Fatalf("want unlocked, got %#v", res.Result)
	}
	var next parameter_explorer.State
	_ = json.Unmarshal(res.StatePatch, &next)
	if next.Checkpoints["n1"] == "" {
		t.Fatal("expected checkpoint timestamp")
	}

	// Forge attempt with a too low — should fail if checkpoint not already set.
	st2 := parameter_explorer.EmptyState()
	st2.Params = map[string]any{"a": 0.5, "b": 0.0, "c": 0.0}
	st2JSON, _ := json.Marshal(st2)
	res2, err := handleParameterExplorerCheckpoint(ActionContext{
		ConfigJSON: cfgJSON,
		StateJSON:  st2JSON,
		Input:      mustJSON(map[string]any{"promptId": "n1", "params": st2.Params}),
	})
	if err != nil {
		t.Fatal(err)
	}
	if res2.Result["unlocked"] == true {
		t.Fatal("forged low a should not unlock")
	}
}

func TestParameterExplorerSubmitAnswer_Filter(t *testing.T) {
	cfg := parameter_explorer.DefaultConfig()
	cfg.NoticingPrompts = []parameter_explorer.NoticingPrompt{
		{ID: "n1", Text: "Describe", Kind: "text", Required: true},
	}
	cfgJSON, _ := json.Marshal(cfg)
	stJSON, _ := json.Marshal(parameter_explorer.EmptyState())

	// Use a mild non-crisis string; filter may or may not block depending on dictionary.
	// Empty answer still ok path.
	res, err := handleParameterExplorerSubmitAnswer(ActionContext{
		ConfigJSON: cfgJSON,
		StateJSON:  stJSON,
		Input:      mustJSON(map[string]any{"promptId": "n1", "answer": "The curve got steeper"}),
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Result["ok"] != true {
		t.Fatalf("got %#v", res.Result)
	}
}

func TestParameterExplorerResetDefaults(t *testing.T) {
	cfg := parameter_explorer.DefaultConfig()
	cfgJSON, _ := json.Marshal(cfg)
	st := parameter_explorer.EmptyState()
	st.Params = map[string]any{"a": 9.0}
	st.Answers = map[string]string{"n1": "x"}
	st.Checkpoints = map[string]string{"n1": "t"}
	st.Trace = []parameter_explorer.TraceEntry{{At: "t", Params: map[string]any{"a": 9.0}}}
	stJSON, _ := json.Marshal(st)

	res, err := handleParameterExplorerResetDefaults(ActionContext{
		ConfigJSON: cfgJSON,
		StateJSON:  stJSON,
	})
	if err != nil {
		t.Fatal(err)
	}
	var next parameter_explorer.State
	_ = json.Unmarshal(res.StatePatch, &next)
	if len(next.Answers) != 0 || len(next.Checkpoints) != 0 || len(next.Trace) != 0 {
		t.Fatalf("expected cleared state, got %#v", next)
	}
	if next.Params["a"] != 1.0 {
		t.Fatalf("expected default a=1, got %#v", next.Params["a"])
	}
}

func TestExtractFreeTextFromState_Answers(t *testing.T) {
	got := ExtractFreeTextFromState([]byte(`{"answers":{"p1":"hello world","p2":""}}`))
	if got != "hello world" {
		t.Fatalf("got %q", got)
	}
}
