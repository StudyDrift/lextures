package contenttools

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/lextures/lextures/server/internal/service/contenttools/tools/explain_it_back"
)

func sampleExplainConfigJSON() json.RawMessage {
	return json.RawMessage(`{
		"prompt":"In your own words, explain why stoichiometry matters in the lab.",
		"minWords":10,
		"maxWords":150,
		"keyPoints":[
			{"id":"kp1","label":"ratio","description":"Mentions mole ratios"},
			{"id":"kp2","label":"balance","description":"Mentions balanced equations"},
			{"id":"kp3","label":"limit","description":"Mentions limiting reactant"}
		],
		"revealKeyPointsAfterSubmit":true,
		"aiFeedback":true,
		"feedbackStyle":"encouraging",
		"attempts":3,
		"includeProbeQuestion":true,
		"allowInstructorNote":true,
		"maxSubmissionsPerDay":10
	}`)
}

func longEnoughText() string {
	return "Stoichiometry uses mole ratios from balanced equations so chemists can predict how much product forms before a reactant runs out in the lab."
}

func TestExplainItBackRegistered(t *testing.T) {
	m := MustDefault().Get(explain_it_back.ID)
	if m == nil {
		t.Fatal("explain_it_back missing from registry")
	}
	if m.AI == nil || m.AI.FeatureID != explain_it_back.FeatureID {
		t.Fatalf("unexpected ai decl: %+v", m.AI)
	}
	if LookupActionHandler(explain_it_back.ID, "submit") == nil ||
		LookupActionHandler(explain_it_back.ID, "instructor_note") == nil ||
		LookupActionHandler(explain_it_back.ID, "test_sample") == nil {
		t.Fatal("handlers missing")
	}
	if err := ValidateConfigJSON(m, sampleExplainConfigJSON()); err != nil {
		t.Fatalf("config invalid: %v", err)
	}
}

func TestExplainItBackSubmitRequiresText(t *testing.T) {
	_, err := handleExplainItBackSubmit(ActionContext{
		Ctx:        context.Background(),
		ConfigJSON: sampleExplainConfigJSON(),
		StateJSON:  json.RawMessage(`{"v":1,"attempts":[]}`),
		Input:      json.RawMessage(`{}`),
	})
	if err == nil || !strings.Contains(err.Error(), "text is required") {
		t.Fatalf("want text required, got %v", err)
	}
}

func TestExplainItBackSubmitTooShort(t *testing.T) {
	res, err := handleExplainItBackSubmit(ActionContext{
		Ctx:        context.Background(),
		ConfigJSON: sampleExplainConfigJSON(),
		StateJSON:  json.RawMessage(`{"v":1,"attempts":[]}`),
		Input:      json.RawMessage(`{"text":"too short"}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Result["error"] != "too_short" {
		t.Fatalf("want too_short, got %#v", res.Result)
	}
}

func TestExplainItBackSubmitReviewWhenAIDisabled(t *testing.T) {
	cfg := sampleExplainConfigJSON()
	// Override aiFeedback false.
	var m map[string]any
	_ = json.Unmarshal(cfg, &m)
	m["aiFeedback"] = false
	raw, _ := json.Marshal(m)

	res, err := handleExplainItBackSubmit(ActionContext{
		Ctx:        context.Background(),
		ConfigJSON: raw,
		StateJSON:  json.RawMessage(`{"v":1,"attempts":[]}`),
		Input:      json.RawMessage(`{"text":` + jsonQuote(longEnoughText()) + `}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != StatusCompleted {
		t.Fatalf("want completed, got %s", res.Status)
	}
	fb, _ := res.Result["feedback"].(explain_it_back.Feedback)
	if fb.Mode != explain_it_back.FeedbackModeReview {
		// Feedback may be returned as map via JSON roundtrip in Result — check mode field.
		if mode, ok := res.Result["mode"].(string); !ok || mode != "review" {
			t.Fatalf("want review mode, got %#v", res.Result)
		}
	}
	var st explain_it_back.State
	if err := json.Unmarshal(res.StatePatch, &st); err != nil {
		t.Fatal(err)
	}
	if len(st.Attempts) != 1 || st.Attempts[0].Feedback == nil || st.Attempts[0].Feedback.Mode != explain_it_back.FeedbackModeReview {
		t.Fatalf("want stored review attempt, got %+v", st.Attempts)
	}
	if st.CompletedAt == "" {
		t.Fatal("completedAt required after first substantive submit")
	}
}

func TestExplainItBackSubmitReviewWhenNoDeps(t *testing.T) {
	res, err := handleExplainItBackSubmit(ActionContext{
		Ctx:        context.Background(),
		ConfigJSON: sampleExplainConfigJSON(),
		StateJSON:  json.RawMessage(`{"v":1,"attempts":[]}`),
		Input:      json.RawMessage(`{"text":` + jsonQuote(longEnoughText()) + `}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	if mode, _ := res.Result["mode"].(string); mode != "review" {
		t.Fatalf("want review fallback without AI deps, got %#v", res.Result)
	}
}

func TestExplainItBackSubmitRateLimited(t *testing.T) {
	st := explain_it_back.State{
		V: 1,
		SubmittedToday: &explain_it_back.SubmittedToday{
			Date:  explain_it_back.TodayUTC(time.Now().UTC()),
			Count: 10,
		},
	}
	raw, _ := json.Marshal(st)
	res, err := handleExplainItBackSubmit(ActionContext{
		Ctx:        context.Background(),
		ConfigJSON: sampleExplainConfigJSON(),
		StateJSON:  raw,
		Input:      json.RawMessage(`{"text":` + jsonQuote(longEnoughText()) + `}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Result["error"] != "rate_limited" {
		t.Fatalf("want rate_limited, got %#v", res.Result)
	}
}

func TestExplainItBackSubmitMaxAttempts(t *testing.T) {
	st := explain_it_back.State{
		V: 1,
		Attempts: []explain_it_back.Attempt{
			{At: "t", Text: "a"},
			{At: "t", Text: "b"},
			{At: "t", Text: "c"},
		},
	}
	raw, _ := json.Marshal(st)
	res, err := handleExplainItBackSubmit(ActionContext{
		Ctx:        context.Background(),
		ConfigJSON: sampleExplainConfigJSON(),
		StateJSON:  raw,
		Input:      json.RawMessage(`{"text":` + jsonQuote(longEnoughText()) + `}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Result["error"] != "max_attempts" {
		t.Fatalf("want max_attempts, got %#v", res.Result)
	}
}

func TestExplainItBackRevisionStoresBothAttempts(t *testing.T) {
	cfg := sampleExplainConfigJSON()
	var m map[string]any
	_ = json.Unmarshal(cfg, &m)
	m["aiFeedback"] = false
	raw, _ := json.Marshal(m)

	res1, err := handleExplainItBackSubmit(ActionContext{
		Ctx:        context.Background(),
		ConfigJSON: raw,
		StateJSON:  json.RawMessage(`{"v":1,"attempts":[]}`),
		Input:      json.RawMessage(`{"text":` + jsonQuote(longEnoughText()) + `}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	res2, err := handleExplainItBackSubmit(ActionContext{
		Ctx:        context.Background(),
		ConfigJSON: raw,
		StateJSON:  res1.StatePatch,
		Input:      json.RawMessage(`{"text":` + jsonQuote(longEnoughText()+" Adding a revision about limiting reactants.") + `}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	var st explain_it_back.State
	if err := json.Unmarshal(res2.StatePatch, &st); err != nil {
		t.Fatal(err)
	}
	if len(st.Attempts) != 2 {
		t.Fatalf("want 2 attempts, got %d", len(st.Attempts))
	}
	if res2.Result["attemptsRemaining"] != 1 {
		t.Fatalf("want 1 attempt left, got %#v", res2.Result["attemptsRemaining"])
	}
}

func TestExplainItBackInstructorNote(t *testing.T) {
	res, err := handleExplainItBackInstructorNote(ActionContext{
		Ctx:          context.Background(),
		ConfigJSON:   sampleExplainConfigJSON(),
		StateJSON:    json.RawMessage(`{"v":1,"attempts":[{"at":"t","text":"x"}]}`),
		Input:        json.RawMessage(`{"text":"Nice start — add the limiting idea."}`),
		InteractRole: "instructor",
	})
	if err != nil {
		t.Fatal(err)
	}
	var st explain_it_back.State
	if err := json.Unmarshal(res.StatePatch, &st); err != nil {
		t.Fatal(err)
	}
	if st.InstructorNote == nil || st.InstructorNote.Text == "" {
		t.Fatalf("want instructor note, got %+v", st.InstructorNote)
	}
}

func TestExplainItBackRedactsKeyPointsFromStudentConfig(t *testing.T) {
	m := MustDefault().Get(explain_it_back.ID)
	cfg := sampleExplainConfigJSON()
	redacted, err := RedactSensitiveConfig(m.ConfigSchema, cfg)
	if err != nil {
		t.Fatal(err)
	}
	var out map[string]any
	if err := json.Unmarshal(redacted, &out); err != nil {
		t.Fatal(err)
	}
	if _, ok := out["keyPoints"]; ok {
		t.Fatalf("keyPoints must be redacted from student payload, got %#v", out)
	}
	if out["prompt"] == nil {
		t.Fatal("prompt should remain")
	}
}

func jsonQuote(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}
