package adminsearchrepo

import (
	"regexp"
	"strings"
	"testing"
)

// goEquivalent converts the Postgres \m / \M word anchors to Go's \b so the
// patterns can be exercised without a database (case-insensitive like ~*).
func goEquivalent(t *testing.T, pattern string) *regexp.Regexp {
	t.Helper()
	p := strings.ReplaceAll(pattern, `\m`, `\b`)
	p = strings.ReplaceAll(p, `\M`, `\b`)
	return regexp.MustCompile(`(?i)` + p)
}

func TestMatchPatterns_ShortQueryNeedsWholeWord(t *testing.T) {
	m := newMatchPatterns("QA")
	if m.fuzzy {
		t.Fatal("2-letter queries should not use fuzzy matching")
	}
	re := goEquivalent(t, m.match)
	for _, s := range []string{"QA Testing 101", "Intro to qa", "C-CPXO86 QA"} {
		if !re.MatchString(s) {
			t.Errorf("expected %q to match QA", s)
		}
	}
	for _, s := range []string{"Telling Qamets", "Aqua", "AQAB"} {
		if re.MatchString(s) {
			t.Errorf("did not expect %q to match QA", s)
		}
	}
}

func TestMatchPatterns_LongerQueryMatchesWordPrefixOnly(t *testing.T) {
	m := newMatchPatterns("bio")
	if !m.fuzzy {
		t.Fatal("3+ letter queries should allow fuzzy matching")
	}
	re := goEquivalent(t, m.match)
	if !re.MatchString("Biology 101") || !re.MatchString("Intro Bio") {
		t.Error("expected word-prefix matches")
	}
	if re.MatchString("Microbiology") {
		t.Error("did not expect a match inside a word")
	}
	word := goEquivalent(t, m.word)
	if word.MatchString("Biology 101") {
		t.Error("whole-word pattern should not match a prefix")
	}
	if !word.MatchString("Intro Bio") {
		t.Error("whole-word pattern should match the whole word")
	}
}

func TestMatchPatterns_EscapesRegexAndNonWordEdges(t *testing.T) {
	m := newMatchPatterns("c++")
	re := goEquivalent(t, m.match)
	if !re.MatchString("Learn C++ today") {
		t.Error("expected literal c++ match")
	}
	if re.MatchString("Learn c today") {
		t.Error("regex metacharacters must be escaped")
	}
	code := goEquivalent(t, newMatchPatterns("C-CPXO86").match)
	if !code.MatchString("C-CPXO86") {
		t.Error("expected course code to match itself")
	}
}
