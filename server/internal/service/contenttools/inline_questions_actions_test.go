package contenttools_test

import (
	"encoding/json"
	"testing"

	"github.com/lextures/lextures/server/internal/service/contenttools"
	"github.com/lextures/lextures/server/internal/service/contenttools/tools/inline_questions"
)

func TestInlineQuestionsSubmitFlow(t *testing.T) {
	reg, err := contenttools.BuildBuiltinRegistry()
	if err != nil {
		t.Fatal(err)
	}
	m := reg.Get(inline_questions.ID)
	if m == nil {
		t.Fatal("missing manifest")
	}

	cv := 3.14159
	cfg := inline_questions.Config{
		Attempts:           2,
		RevealCorrectAfter: inline_questions.RevealLastAttempt,
		ScorePolicy:        inline_questions.ScoreBest,
		Questions: []inline_questions.Question{
			{
				ID:     "q1",
				Type:   inline_questions.TypeSingle,
				Prompt: "Cap?",
				Options: []inline_questions.Option{
					{ID: "a", Text: "A", Correct: false, Feedback: "no"},
					{ID: "b", Text: "B", Correct: true, Feedback: "yes"},
				},
				Explanation: "because",
			},
			{
				ID:           "q2",
				Type:         inline_questions.TypeNumeric,
				Prompt:       "pi?",
				CorrectValue: &cv,
				Tolerance:    &inline_questions.Tolerance{Kind: inline_questions.ToleranceAbsolute, Value: 0.05},
			},
		},
	}
	cfgJSON, _ := json.Marshal(cfg)
	stJSON, _ := json.Marshal(inline_questions.EmptyState())

	in, _ := json.Marshal(map[string]any{"questionId": "q1", "value": "a"})
	res, err := contenttools.DispatchAction(m, "submit", contenttools.ActionContext{
		ConfigJSON: cfgJSON,
		StateJSON:  stJSON,
		Input:      in,
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Result["correct"] != false {
		t.Fatalf("want incorrect: %#v", res.Result)
	}
	if res.Result["feedback"] != "no" {
		t.Fatalf("feedback: %#v", res.Result["feedback"])
	}
	if _, ok := res.Result["correctAnswer"]; ok {
		t.Fatal("correctAnswer should be absent before reveal")
	}

	in2, _ := json.Marshal(map[string]any{"questionId": "q1", "value": "b"})
	res2, err := contenttools.DispatchAction(m, "submit", contenttools.ActionContext{
		ConfigJSON: cfgJSON,
		StateJSON:  res.StatePatch,
		Input:      in2,
	})
	if err != nil {
		t.Fatal(err)
	}
	if res2.Result["correct"] != true {
		t.Fatalf("want correct: %#v", res2.Result)
	}
	if res2.Result["explanation"] != "because" {
		t.Fatalf("explanation: %#v", res2.Result["explanation"])
	}

	// Exhaust q2 then refuse.
	in3, _ := json.Marshal(map[string]any{"questionId": "q2", "value": 1.0})
	res3, err := contenttools.DispatchAction(m, "submit", contenttools.ActionContext{
		ConfigJSON: cfgJSON,
		StateJSON:  res2.StatePatch,
		Input:      in3,
	})
	if err != nil {
		t.Fatal(err)
	}
	in4, _ := json.Marshal(map[string]any{"questionId": "q2", "value": 2.0})
	res4, err := contenttools.DispatchAction(m, "submit", contenttools.ActionContext{
		ConfigJSON: cfgJSON,
		StateJSON:  res3.StatePatch,
		Input:      in4,
	})
	if err != nil {
		t.Fatal(err)
	}
	in5, _ := json.Marshal(map[string]any{"questionId": "q2", "value": 3.14})
	res5, err := contenttools.DispatchAction(m, "submit", contenttools.ActionContext{
		ConfigJSON: cfgJSON,
		StateJSON:  res4.StatePatch,
		Input:      in5,
	})
	if err != nil {
		t.Fatal(err)
	}
	if res5.Result["error"] != "max_attempts" {
		t.Fatalf("want max_attempts, got %#v", res5.Result)
	}
	if string(res5.StatePatch) != "" && len(res5.StatePatch) > 0 {
		// max_attempts should leave state unchanged (no patch).
		t.Fatalf("expected no state patch on max_attempts, got %s", res5.StatePatch)
	}

	// Numeric within tolerance on a fresh question.
	fresh := inline_questions.EmptyState()
	fresh.Answers["q1"] = inline_questions.QuestionAnswer{
		Attempts: []inline_questions.Attempt{{Value: "b", Correct: true, Points: 1}},
	}
	freshJSON, _ := json.Marshal(fresh)
	inPi, _ := json.Marshal(map[string]any{"questionId": "q2", "value": 3.14})
	resPi, err := contenttools.DispatchAction(m, "submit", contenttools.ActionContext{
		ConfigJSON: cfgJSON,
		StateJSON:  freshJSON,
		Input:      inPi,
	})
	if err != nil {
		t.Fatal(err)
	}
	if resPi.Result["correct"] != true {
		t.Fatalf("3.14 should be correct: %#v", resPi.Result)
	}
}
