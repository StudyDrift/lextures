package contenttools_test

import (
	"encoding/json"
	"testing"

	"github.com/lextures/lextures/server/internal/service/contenttools"
	"github.com/lextures/lextures/server/internal/service/contenttools/tools/sort_sequence"
)

func sampleSortConfig() (json.RawMessage, sort_sequence.Config) {
	cfg := sort_sequence.Config{
		Mode:   sort_sequence.ModeCategorize,
		Prompt: "Sort",
		Items: []sort_sequence.Item{
			{ID: "a", Text: "Acid"},
			{ID: "b", Text: "Base"},
		},
		Buckets: []sort_sequence.Bucket{
			{ID: "acid", Label: "Acid"},
			{ID: "base", Label: "Base"},
		},
		CorrectBucketByItem: map[string]json.RawMessage{
			"a": json.RawMessage(`"acid"`),
			"b": json.RawMessage(`"base"`),
		},
		ItemFeedback: map[string]string{
			"a": "Acids donate protons.",
		},
		Attempts:               2,
		ShowPerItemCorrectness: true,
		LockCorrect:            true,
		ScoreMode:              sort_sequence.ScorePerItem,
		ShuffleItems:           true,
	}
	raw, _ := json.Marshal(cfg)
	return raw, cfg
}

func TestSortSequenceCheckAndLock(t *testing.T) {
	reg, err := contenttools.BuildBuiltinRegistry()
	if err != nil {
		t.Fatalf("registry: %v", err)
	}
	m := reg.Get(sort_sequence.ID)
	if m == nil {
		t.Fatal("sort_sequence missing from registry")
	}
	cfgJSON, _ := sampleSortConfig()
	stJSON, _ := json.Marshal(sort_sequence.EmptyState())

	wrongPlace := sort_sequence.MarshalCategorizePlacement(map[string]*string{
		"a": ssStr("base"),
		"b": ssStr("base"),
	})
	in, _ := json.Marshal(map[string]any{"placement": json.RawMessage(wrongPlace)})
	res, err := contenttools.DispatchAction(m, "check", contenttools.ActionContext{
		ConfigJSON: cfgJSON,
		StateJSON:  stJSON,
		Input:      in,
	})
	if err != nil {
		t.Fatalf("check: %v", err)
	}
	if res.Result["scorePct"].(float64) != 50 {
		t.Fatalf("scorePct: %v", res.Result["scorePct"])
	}
	perItem, _ := res.Result["perItem"].(map[string]any)
	aRes, _ := perItem["a"].(map[string]any)
	if aRes["correct"] != false {
		t.Fatalf("a should be wrong: %v", aRes)
	}
	if aRes["feedback"] != "Acids donate protons." {
		t.Fatalf("feedback: %v", aRes["feedback"])
	}

	var st sort_sequence.State
	_ = json.Unmarshal(res.StatePatch, &st)
	if len(st.Attempts) != 1 {
		t.Fatalf("attempts: %d", len(st.Attempts))
	}
	if len(st.LockedItemIDs) != 1 || st.LockedItemIDs[0] != "b" {
		t.Fatalf("locked: %v", st.LockedItemIDs)
	}

	// Reset attempt returns unlocked to tray; locked stays.
	res2, err := contenttools.DispatchAction(m, "reset_attempt", contenttools.ActionContext{
		ConfigJSON: cfgJSON,
		StateJSON:  res.StatePatch,
	})
	if err != nil {
		t.Fatalf("reset: %v", err)
	}
	var st2 sort_sequence.State
	_ = json.Unmarshal(res2.StatePatch, &st2)
	place, ok := sort_sequence.ParseCategorizePlacement(st2.Placement)
	if !ok {
		t.Fatal("parse placement")
	}
	if place["b"] == nil || *place["b"] != "base" {
		t.Fatalf("locked b should remain: %v", place["b"])
	}
	if place["a"] != nil {
		t.Fatalf("a should be tray: %v", place["a"])
	}
	if len(st2.Attempts) != 1 {
		t.Fatal("reset must not clear attempts")
	}

	// Second check with correct placement.
	right := sort_sequence.MarshalCategorizePlacement(map[string]*string{
		"a": ssStr("acid"),
		"b": ssStr("base"),
	})
	in2, _ := json.Marshal(map[string]any{"placement": json.RawMessage(right)})
	res3, err := contenttools.DispatchAction(m, "check", contenttools.ActionContext{
		ConfigJSON: cfgJSON,
		StateJSON:  res2.StatePatch,
		Input:      in2,
	})
	if err != nil {
		t.Fatalf("check2: %v", err)
	}
	if res3.Result["scorePct"].(float64) != 100 {
		t.Fatalf("score: %v", res3.Result["scorePct"])
	}
	if res3.Status != contenttools.StatusCompleted {
		t.Fatalf("status: %s", res3.Status)
	}

	// Exhaustion.
	stJSON3 := res3.StatePatch
	res4, err := contenttools.DispatchAction(m, "check", contenttools.ActionContext{
		ConfigJSON: cfgJSON,
		StateJSON:  stJSON3,
		Input:      in2,
	})
	if err != nil {
		t.Fatalf("check3: %v", err)
	}
	if res4.Result["error"] != "max_attempts" {
		t.Fatalf("expected max_attempts, got %v", res4.Result)
	}
}

func TestSortSequenceOrderTieAndRedaction(t *testing.T) {
	reg, err := contenttools.BuildBuiltinRegistry()
	if err != nil {
		t.Fatalf("registry: %v", err)
	}
	m := reg.Get(sort_sequence.ID)
	cfg := sort_sequence.Config{
		Mode:                   sort_sequence.ModeOrder,
		Prompt:                 "Order",
		Items:                  []sort_sequence.Item{{ID: "a", Text: "A"}, {ID: "b", Text: "B"}, {ID: "c", Text: "C"}},
		CorrectOrder:           []string{"a", "b", "c"},
		TieGroups:              [][]string{{"a", "b"}},
		Attempts:               3,
		ShowPerItemCorrectness: true,
		LockCorrect:            false,
		ScoreMode:              sort_sequence.ScorePerItem,
	}
	cfgJSON, _ := json.Marshal(cfg)
	stJSON, _ := json.Marshal(sort_sequence.EmptyState())
	place := sort_sequence.MarshalOrderPlacement([]string{"b", "a", "c"})
	in, _ := json.Marshal(map[string]any{"placement": json.RawMessage(place)})
	res, err := contenttools.DispatchAction(m, "check", contenttools.ActionContext{
		ConfigJSON: cfgJSON,
		StateJSON:  stJSON,
		Input:      in,
	})
	if err != nil {
		t.Fatalf("check: %v", err)
	}
	if res.Result["scorePct"].(float64) != 100 {
		t.Fatalf("tie order should score 100: %v", res.Result["scorePct"])
	}

	// Redaction: student config must not include answer keys.
	redacted, err := contenttools.RedactSensitiveConfig(m.ConfigSchema, cfgJSON)
	if err != nil {
		t.Fatalf("redact: %v", err)
	}
	var cfgOut map[string]any
	_ = json.Unmarshal(redacted, &cfgOut)
	if _, ok := cfgOut["correctOrder"]; ok {
		t.Fatal("correctOrder must be redacted")
	}
	if _, ok := cfgOut["tieGroups"]; ok {
		t.Fatal("tieGroups must be redacted")
	}
	if _, ok := cfgOut["itemFeedback"]; ok {
		t.Fatal("itemFeedback must be redacted")
	}
}

func ssStr(s string) *string { return &s }
