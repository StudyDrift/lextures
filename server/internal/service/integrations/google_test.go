package integrations

import (
	"reflect"
	"testing"
)

func TestComputeRosterDiff(t *testing.T) {
	external := []ExternalMember{
		{Email: "Alice@example.com", FullName: "Alice", Role: "student"},
		{Email: "bob@example.com", FullName: "Bob", Role: "student"},
		{Email: "", FullName: "no email", Role: "student"}, // skipped
		{Email: "teach@example.com", FullName: "Teach", Role: "teacher"},
	}
	current := []string{"bob@example.com", "carol@example.com"}

	diff := ComputeRosterDiff(external, current)

	if len(diff.Added) != 2 {
		t.Fatalf("Added = %d, want 2 (%+v)", len(diff.Added), diff.Added)
	}
	addedEmails := map[string]bool{}
	for _, m := range diff.Added {
		addedEmails[m.Email] = true
	}
	if !addedEmails["Alice@example.com"] || !addedEmails["teach@example.com"] {
		t.Errorf("unexpected Added set: %+v", diff.Added)
	}
	if len(diff.Unchanged) != 1 || diff.Unchanged[0].Email != "bob@example.com" {
		t.Errorf("Unchanged = %+v, want [bob]", diff.Unchanged)
	}
	if !reflect.DeepEqual(diff.Removed, []string{"carol@example.com"}) {
		t.Errorf("Removed = %+v, want [carol]", diff.Removed)
	}
}

func TestComputeRosterDiffEmpty(t *testing.T) {
	diff := ComputeRosterDiff(nil, nil)
	if len(diff.Added) != 0 || len(diff.Unchanged) != 0 || len(diff.Removed) != 0 {
		t.Errorf("empty diff should have empty slices, got %+v", diff)
	}
	// Slices must be non-nil so they serialize as [] not null.
	if diff.Added == nil || diff.Unchanged == nil || diff.Removed == nil {
		t.Error("diff slices must be non-nil")
	}
}
