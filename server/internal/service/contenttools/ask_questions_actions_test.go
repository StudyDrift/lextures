package contenttools

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/lextures/lextures/server/internal/service/aiprovider"
	"github.com/lextures/lextures/server/internal/service/contenttools/tools/ask_questions"
)

func TestAskQuestionsRegistered(t *testing.T) {
	m := MustDefault().Get(ask_questions.ID)
	if m == nil {
		t.Fatal("ask_questions missing from registry")
	}
	if m.AI == nil || m.AI.FeatureID != ask_questions.FeatureID {
		t.Fatalf("unexpected ai decl: %+v", m.AI)
	}
	if LookupActionHandler(ask_questions.ID, "ask") == nil || LookupActionHandler(ask_questions.ID, "clear") == nil {
		t.Fatal("ask/clear handlers missing")
	}
	if err := ValidateConfigJSON(m, json.RawMessage(`{"stance":"explain"}`)); err != nil {
		t.Fatalf("default config invalid: %v", err)
	}
}

func TestAskQuestionsClearAction(t *testing.T) {
	st := ask_questions.State{
		V: 1,
		Turns: []ask_questions.Turn{
			{ID: "1", Role: "user", Text: "hi", CreatedAt: "t"},
		},
	}
	raw, _ := json.Marshal(st)
	res, err := handleAskQuestionsClear(ActionContext{
		Ctx:       context.Background(),
		StateJSON: raw,
		// Pool nil skips course settings gate for unit tests.
	})
	if err != nil {
		t.Fatal(err)
	}
	var next ask_questions.State
	if err := json.Unmarshal(res.StatePatch, &next); err != nil {
		t.Fatal(err)
	}
	if len(next.Turns) != 0 {
		t.Fatalf("want empty turns, got %d", len(next.Turns))
	}
}

func TestAskQuestionsAskRequiresQuestion(t *testing.T) {
	_, err := handleAskQuestionsAsk(ActionContext{
		Ctx:       context.Background(),
		StateJSON: json.RawMessage(`{"v":1,"turns":[]}`),
		Input:     json.RawMessage(`{}`),
	})
	if err == nil || !strings.Contains(err.Error(), "question is required") {
		t.Fatalf("want question required, got %v", err)
	}
}

func TestAskQuestionsAskRateLimited(t *testing.T) {
	st := ask_questions.State{
		V: 1,
		AskedToday: &ask_questions.AskedToday{
			Date:  ask_questions.TodayUTC(time.Now().UTC()),
			Count: 20,
		},
	}
	raw, _ := json.Marshal(st)
	res, err := handleAskQuestionsAsk(ActionContext{
		Ctx:        context.Background(),
		ConfigJSON: json.RawMessage(`{"maxQuestionsPerDay":20}`),
		StateJSON:  raw,
		Input:      json.RawMessage(`{"question":"What is this?"}`),
		Completer:  aiprovider.DryRunToolCallingCompleter{},
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Result["error"] != "rate_limited" {
		t.Fatalf("want rate_limited, got %#v", res.Result)
	}
	if len(res.StatePatch) > 0 {
		t.Fatal("rate limit must not mutate state")
	}
}

func TestAskQuestionsAskRequiresAIDeps(t *testing.T) {
	res, err := handleAskQuestionsAsk(ActionContext{
		Ctx:       context.Background(),
		StateJSON: json.RawMessage(`{"v":1,"turns":[]}`),
		Input:     json.RawMessage(`{"question":"What is stoichiometric?"}`),
	})
	if err == nil || !strings.Contains(err.Error(), "AI runtime deps") {
		t.Fatalf("want AI deps error, got res=%v err=%v", res, err)
	}
}
