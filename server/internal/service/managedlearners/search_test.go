package managedlearners

import "testing"

func TestContainsLikePatternEscapesWildcards(t *testing.T) {
	got := containsLikePattern(`100%_done\`)
	want := `%100\%\_done\\%`
	if got != want {
		t.Fatalf("pattern %q, want %q", got, want)
	}
}
