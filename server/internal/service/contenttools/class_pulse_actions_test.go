package contenttools_test

import (
	"encoding/json"
	"testing"

	"github.com/lextures/lextures/server/internal/service/contenttools"
	"github.com/lextures/lextures/server/internal/service/contenttools/tools/class_pulse"
)

func sampleClassPulseConfig(overrides map[string]any) (json.RawMessage, class_pulse.Config) {
	cfg := class_pulse.Config{
		Question:        "Which approach is best?",
		Options:         []class_pulse.Option{{ID: "a", Text: "A"}, {ID: "b", Text: "B"}, {ID: "c", Text: "C"}},
		CorrectOptionID: "a",
		Explanation:     "A is correct because…",
		AllowSecondVote: true,
		RevealCorrect:   class_pulse.RevealAfterRevote,
		MinRespondents:  5,
		ScopeToSection:  false,
		ShowPercentages: true,
	}
	raw, _ := json.Marshal(cfg)
	if len(overrides) > 0 {
		var m map[string]any
		_ = json.Unmarshal(raw, &m)
		for k, v := range overrides {
			m[k] = v
		}
		raw, _ = json.Marshal(m)
		cfg = class_pulse.ParseConfig(raw)
	}
	return raw, cfg
}

func TestClassPulseRedactionAndVoteGate(t *testing.T) {
	reg, err := contenttools.BuildBuiltinRegistry()
	if err != nil {
		t.Fatal(err)
	}
	m := reg.Get(class_pulse.ID)
	if m == nil {
		t.Fatal("missing class_pulse manifest")
	}

	cfgJSON, _ := sampleClassPulseConfig(nil)
	redacted, err := contenttools.RedactSensitiveConfig(m.ConfigSchema, cfgJSON)
	if err != nil {
		t.Fatal(err)
	}
	var red map[string]any
	_ = json.Unmarshal(redacted, &red)
	if _, ok := red["correctOptionId"]; ok {
		t.Fatalf("correctOptionId leaked: %s", redacted)
	}
	if _, ok := red["explanation"]; ok {
		t.Fatalf("explanation leaked: %s", redacted)
	}
	opts, _ := red["options"].([]any)
	if len(opts) != 3 {
		t.Fatalf("options: %#v", opts)
	}

	stJSON, _ := json.Marshal(class_pulse.EmptyState())

	// Aggregate before vote denied.
	res, err := contenttools.DispatchAction(m, "aggregate", contenttools.ActionContext{
		ConfigJSON:   cfgJSON,
		StateJSON:    stJSON,
		InteractRole: "student",
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Result["error"] != "vote_required" {
		t.Fatalf("want vote_required, got %#v", res.Result)
	}

	in, _ := json.Marshal(map[string]any{"optionId": "b", "round": 1})
	voted, err := contenttools.DispatchAction(m, "vote", contenttools.ActionContext{
		ConfigJSON:   cfgJSON,
		StateJSON:    stJSON,
		Input:        in,
		InteractRole: "student",
	})
	if err != nil {
		t.Fatal(err)
	}
	if voted.Result["error"] != nil {
		t.Fatalf("vote error: %#v", voted.Result)
	}
	agg, ok := voted.Result["aggregate"].(class_pulse.RoundAggregate)
	if !ok {
		// JSON round-trip via map
		rawAgg, _ := json.Marshal(voted.Result["aggregate"])
		if err := json.Unmarshal(rawAgg, &agg); err != nil {
			t.Fatalf("aggregate missing: %#v", voted.Result)
		}
	}
	if !agg.Suppressed || agg.Reason != "small_n" {
		t.Fatalf("want small_n suppression, got %#v", agg)
	}
	if voted.Result["reveal"] != nil {
		t.Fatalf("reveal must wait for revote: %#v", voted.Result["reveal"])
	}
	var st class_pulse.State
	if err := json.Unmarshal(voted.StatePatch, &st); err != nil {
		t.Fatal(err)
	}
	if !st.HasVotedRound(1) || st.Votes[0].OptionID != "b" {
		t.Fatalf("state: %#v", st)
	}

	// Double vote refused.
	again, err := contenttools.DispatchAction(m, "vote", contenttools.ActionContext{
		ConfigJSON:   cfgJSON,
		StateJSON:    voted.StatePatch,
		Input:        in,
		InteractRole: "student",
	})
	if err != nil {
		t.Fatal(err)
	}
	if again.Result["error"] != "already_voted" {
		t.Fatalf("want already_voted, got %#v", again.Result)
	}
	var st2 class_pulse.State
	_ = json.Unmarshal(voted.StatePatch, &st2)
	if st2.Votes[0].OptionID != "b" {
		t.Fatal("stored vote changed")
	}

	// Revote reveals correct.
	in2, _ := json.Marshal(map[string]any{"optionId": "a", "round": 2})
	revoted, err := contenttools.DispatchAction(m, "vote", contenttools.ActionContext{
		ConfigJSON:   cfgJSON,
		StateJSON:    voted.StatePatch,
		Input:        in2,
		InteractRole: "student",
	})
	if err != nil {
		t.Fatal(err)
	}
	reveal, _ := revoted.Result["reveal"].(map[string]any)
	if reveal == nil || reveal["correctOptionId"] != "a" {
		t.Fatalf("reveal missing: %#v", revoted.Result)
	}
	if revoted.Result["aggregateRound2"] == nil {
		t.Fatalf("round2 aggregate missing: %#v", revoted.Result)
	}
}

func TestClassPulseInstructorExcludedFromAggregate(t *testing.T) {
	reg, err := contenttools.BuildBuiltinRegistry()
	if err != nil {
		t.Fatal(err)
	}
	m := reg.Get(class_pulse.ID)
	cfgJSON, _ := sampleClassPulseConfig(map[string]any{"minRespondents": 3, "allowSecondVote": false, "revealCorrect": "never"})

	// Simulate five student-less path: instructor vote alone stays suppressed / learners=0 in unit (no pool).
	stJSON, _ := json.Marshal(class_pulse.EmptyState())
	in, _ := json.Marshal(map[string]any{"optionId": "a", "round": 1})
	res, err := contenttools.DispatchAction(m, "vote", contenttools.ActionContext{
		ConfigJSON:   cfgJSON,
		StateJSON:    stJSON,
		Input:        in,
		InteractRole: "instructor",
	})
	if err != nil {
		t.Fatal(err)
	}
	rawAgg, _ := json.Marshal(res.Result["aggregate"])
	var agg class_pulse.RoundAggregate
	_ = json.Unmarshal(rawAgg, &agg)
	// Without DB, self is appended with instructor role and excluded → 0 learners → suppressed.
	if agg.Learners != 0 || !agg.Suppressed {
		t.Fatalf("instructor vote must not count: %#v", agg)
	}
}

func TestGuardClassPulseStatePut(t *testing.T) {
	cur, _ := json.Marshal(class_pulse.State{Votes: []class_pulse.Vote{{Round: 1, OptionID: "a", At: "t"}}})
	next, _ := json.Marshal(class_pulse.State{Votes: []class_pulse.Vote{{Round: 1, OptionID: "b", At: "t"}}})
	blocked, msg := contenttools.GuardClassPulseStatePut(class_pulse.ID, cur, next)
	if !blocked || msg == "" {
		t.Fatal("expected block")
	}
	if blocked, _ := contenttools.GuardClassPulseStatePut("other", cur, next); blocked {
		t.Fatal("other tools unaffected")
	}
}
