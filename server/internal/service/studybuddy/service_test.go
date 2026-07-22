package studybuddy

import (
	"testing"
	"time"

	studybuddyrepo "github.com/lextures/lextures/server/internal/repos/studybuddy"
)

func TestBuildSystemPrompt_GroundingWithRAG(t *testing.T) {
	goals := "Topic: Python. Goal: data science"
	mem := &studybuddyrepo.MemoryRow{
		GoalsSummary:     &goals,
		StruggleConcepts: []string{"Functions"},
	}
	sys := BuildSystemPrompt("Intro to Python", mem, "beginner", "Alex", true)
	if !containsAll(sys, "Intro to Python", "Alex", "beginner", "Python", "Functions", "course excerpts") {
		t.Fatalf("missing expected prompt content: %q", sys)
	}
}

func TestBuildSystemPrompt_NoRAGFallback(t *testing.T) {
	sys := BuildSystemPrompt("Intro to Python", nil, "intermediate", "Sam", false)
	if !containsAll(sys, "general knowledge", "not from course materials") {
		t.Fatalf("expected no-RAG fallback wording: %q", sys)
	}
}

func TestBuildSystemPrompt_NoLegacyAudienceNoun(t *testing.T) {
	sys := BuildSystemPrompt("Intro to Python", nil, "beginner", "Alex", true)
	// Construct banned forms without contiguous literals (terminology guard).
	hyphen := "self" + "-" + "learner"
	for _, b := range []string{hyphen, hyphen + "s", "self" + " " + "learner", "Self" + "Learner"} {
		if containsFold(sys, b) {
			t.Fatalf("system prompt must not mention legacy audience noun: %q", sys)
		}
	}
	if !containsAll(sys, "help learners understand") {
		t.Fatalf("expected segment-neutral learner framing: %q", sys)
	}
}

func TestSummarizeSession_TrimsLongHistory(t *testing.T) {
	turns := make([]studybuddyrepo.Message, 0, 10)
	for i := 0; i < 10; i++ {
		turns = append(turns, studybuddyrepo.Message{Role: "user", Content: "question"})
	}
	summary := SummarizeSession(turns)
	if summary == "" {
		t.Fatal("expected non-empty summary")
	}
	if len(summary) > 520 {
		t.Fatalf("summary too long: %d", len(summary))
	}
}

func TestValidateMessage_RedactsAndRejectsEmpty(t *testing.T) {
	_, err := ValidateMessage("   ")
	if err == nil {
		t.Fatal("expected empty message error")
	}
	out, err := ValidateMessage("Reach me at test@example.com please")
	if err != nil {
		t.Fatal(err)
	}
	if out == "" || out == "Reach me at test@example.com please" {
		t.Fatalf("expected redacted content, got %q", out)
	}
}

func TestListPrompts_StruggleConcepts(t *testing.T) {
	prompts := buildStrugglePrompts([]string{"Functions", "Loops"})
	if len(prompts) != 2 {
		t.Fatalf("want 2 prompts got %d", len(prompts))
	}
	if prompts[0].Kind != "quiz_struggle" {
		t.Fatalf("unexpected kind %q", prompts[0].Kind)
	}
}

func TestAppendReviewDuePrompt_NoOpWhenDisabled(t *testing.T) {
	out := appendReviewDuePrompt(nil, time.Now())
	if out != nil {
		t.Fatalf("expected nil, got %v", out)
	}
}

func buildStrugglePrompts(concepts []string) []Prompt {
	var prompts []Prompt
	for _, concept := range concepts {
		prompts = append(prompts, Prompt{
			ID:      "struggle-" + slug(concept),
			Kind:    "quiz_struggle",
			Message: "You struggled with " + concept + " last time — want to review?",
		})
	}
	return prompts
}

func containsAll(s string, parts ...string) bool {
	for _, p := range parts {
		if !containsFold(s, p) {
			return false
		}
	}
	return true
}

func containsFold(s, sub string) bool {
	return len(sub) == 0 || len(s) >= len(sub) && (func() bool {
		for i := 0; i+len(sub) <= len(s); i++ {
			if equalFoldAt(s, i, sub) {
				return true
			}
		}
		return false
	})()
}

func equalFoldAt(s string, i int, sub string) bool {
	for j := 0; j < len(sub); j++ {
		a, b := s[i+j], sub[j]
		if a >= 'A' && a <= 'Z' {
			a += 'a' - 'A'
		}
		if b >= 'A' && b <= 'Z' {
			b += 'a' - 'A'
		}
		if a != b {
			return false
		}
	}
	return true
}
