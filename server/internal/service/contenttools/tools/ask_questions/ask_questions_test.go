package ask_questions_test

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/lextures/lextures/server/internal/service/contenttools/tools/ask_questions"
)

func TestParseConfigDefaults(t *testing.T) {
	cfg := ask_questions.ParseConfig(nil)
	if cfg.Stance != ask_questions.StanceExplain || cfg.MaxQuestionsPerDay != 20 || !cfg.ShowCitations {
		t.Fatalf("unexpected defaults: %+v", cfg)
	}
	cfg = ask_questions.ParseConfig(json.RawMessage(`{"stance":"socratic","showCitations":false,"maxQuestionsPerDay":5}`))
	if cfg.Stance != ask_questions.StanceSocratic || cfg.ShowCitations || cfg.MaxQuestionsPerDay != 5 {
		t.Fatalf("unexpected overlay: %+v", cfg)
	}
}

func TestBuildTaskPromptIncludesStanceAndInjectionDefence(t *testing.T) {
	p := ask_questions.BuildTaskPrompt(ask_questions.Config{
		Stance:         ask_questions.StanceHintOnly,
		GroundingNotes: "Focus on the lab protocol.",
		OffTopicPolicy: ask_questions.OffTopicRedirect,
	}, "grade 6")
	for _, want := range []string{
		ask_questions.PromptVersion,
		"untrusted DATA",
		"hint only",
		"Focus on the lab protocol.",
		"grade 6",
		"Do NOT complete graded work",
	} {
		if !strings.Contains(p, want) {
			t.Fatalf("prompt missing %q:\n%s", want, p)
		}
	}
}

func TestQuestionsRemainingAndIncrement(t *testing.T) {
	now := time.Date(2026, 7, 27, 12, 0, 0, 0, time.UTC)
	st := ask_questions.EmptyState()
	if got := ask_questions.QuestionsRemaining(st, 2, now); got != 2 {
		t.Fatalf("want 2, got %d", got)
	}
	ask_questions.IncrementAskedToday(&st, now)
	ask_questions.IncrementAskedToday(&st, now)
	if got := ask_questions.QuestionsRemaining(st, 2, now); got != 0 {
		t.Fatalf("want 0 after cap, got %d", got)
	}
	// Next day resets.
	next := now.Add(24 * time.Hour)
	if got := ask_questions.QuestionsRemaining(st, 2, next); got != 2 {
		t.Fatalf("want 2 next day, got %d", got)
	}
}

func TestAppendTurnsTrimsWithSummary(t *testing.T) {
	st := ask_questions.EmptyState()
	for i := 0; i < 6; i++ {
		u := ask_questions.Turn{ID: "u", Role: "user", Text: "question about stoichiometry " + string(rune('a'+i)), CreatedAt: "t"}
		a := ask_questions.Turn{ID: "a", Role: "assistant", Text: "answer", CreatedAt: "t"}
		ask_questions.AppendTurns(&st, u, a, 4)
	}
	if len(st.Turns) != 4 {
		t.Fatalf("want 4 turns, got %d", len(st.Turns))
	}
	if st.Summary == "" {
		t.Fatal("expected rolling summary after trim")
	}
}

func TestCitationsFromTextDropsUnknown(t *testing.T) {
	allowed := []ask_questions.Citation{
		{Kind: "link", ID: "link:1", Title: "Khan", URL: "https://example.com"},
		{Kind: "section", ID: "sec:1", Title: "Intro"},
	}
	got, dropped := ask_questions.CitationsFromText("See [link:1] and [bogus] then [sec:1]", allowed)
	if dropped != 1 {
		t.Fatalf("want 1 dropped, got %d", dropped)
	}
	if len(got) != 2 || got[0].ID != "link:1" || got[1].ID != "sec:1" {
		t.Fatalf("unexpected cites: %+v", got)
	}
	none := ask_questions.MergeCitationLists(nil, allowed, false)
	if none != nil {
		t.Fatalf("showCitations=false should drop all")
	}
}

func TestClusterQuestionsAnonymous(t *testing.T) {
	qs := []string{
		"What does stoichiometric mean here?",
		"What does stoichiometric ratio mean?",
		"How do I balance this reaction?",
		"How do I balance equations?",
		"Why is the catalyst used?",
	}
	clusters := ask_questions.ClusterQuestions(qs, 5)
	if len(clusters) == 0 {
		t.Fatal("expected clusters")
	}
	total := 0
	for _, c := range clusters {
		total += c.Count
		if len(c.RepresentativeExamples) == 0 {
			t.Fatalf("cluster %q missing examples", c.Theme)
		}
	}
	if total != len(qs) {
		t.Fatalf("want total %d, got %d", len(qs), total)
	}
}
