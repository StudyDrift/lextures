package translationmemory

import "testing"

func TestSourceHashStable(t *testing.T) {
	h1 := SourceHash("Hello  World")
	h2 := SourceHash("hello world")
	if h1 != h2 {
		t.Fatalf("expected normalized hash match, got %q vs %q", h1, h2)
	}
}

func TestTrigramSimilarity(t *testing.T) {
	same := TrigramSimilarity("The water cycle moves water", "The water cycle moves water")
	if same < 0.99 {
		t.Fatalf("identical: got %v", same)
	}
	close := TrigramSimilarity(
		"The water cycle moves water through the environment",
		"The water cycle moves water through the earth",
	)
	if close < 0.5 {
		t.Fatalf("similar sentences expected >0.5, got %v", close)
	}
	diff := TrigramSimilarity("Hello", "Completely different topic")
	if diff > 0.3 {
		t.Fatalf("unrelated expected low score, got %v", diff)
	}
}

func TestFindGlossaryMatches(t *testing.T) {
	entries := []GlossaryEntry{{SourceTerm: "assignment", TargetTerm: "tarea"}}
	matches := FindGlossaryMatches("Complete the assignment by Friday.", entries)
	if len(matches) != 1 {
		t.Fatalf("expected 1 match, got %d", len(matches))
	}
	if matches[0].TargetTerm != "tarea" {
		t.Fatalf("got target %q", matches[0].TargetTerm)
	}
}

func TestSuggestGlossaryTranslation(t *testing.T) {
	entries := []GlossaryEntry{{SourceTerm: "assignment", TargetTerm: "tarea"}}
	got := SuggestGlossaryTranslation("Submit your assignment today.", entries)
	if got != "Submit your tarea today." {
		t.Fatalf("got %q", got)
	}
}

func TestPrefixWords(t *testing.T) {
	if got := PrefixWords("one two three four", 3); got != "one two three" {
		t.Fatalf("got %q", got)
	}
}
