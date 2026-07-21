package parentassign

import (
	"testing"
)

func TestValidateGuardians(t *testing.T) {
	t.Parallel()
	if err := ValidateGuardians(nil); err == nil {
		t.Fatal("expected error for empty")
	}
	ok := []GuardianInput{{Name: "Pat Parent", Email: "pat@example.com"}}
	if err := ValidateGuardians(ok); err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	badEmail := []GuardianInput{{Name: "Pat", Email: "not-an-email"}}
	if err := ValidateGuardians(badEmail); err == nil {
		t.Fatal("expected invalid email")
	}
	dup := []GuardianInput{
		{Name: "A", Email: "a@example.com"},
		{Name: "B", Email: "A@example.com"},
	}
	if err := ValidateGuardians(dup); err == nil {
		t.Fatal("expected duplicate email")
	}
	four := []GuardianInput{
		{Name: "1", Email: "1@example.com"},
		{Name: "2", Email: "2@example.com"},
		{Name: "3", Email: "3@example.com"},
		{Name: "4", Email: "4@example.com"},
	}
	if err := ValidateGuardians(four); err == nil {
		t.Fatal("expected max 3")
	}
	badRel := []GuardianInput{{Name: "Pat", Email: "pat@example.com", Relationship: "cousin"}}
	if err := ValidateGuardians(badRel); err == nil {
		t.Fatal("expected bad relationship")
	}
}

func TestNormalizeRelationship(t *testing.T) {
	t.Parallel()
	if got := normalizeRelationship(""); got != "parent" {
		t.Fatalf("got %q", got)
	}
	if got := normalizeRelationship(" Guardian "); got != "guardian" {
		t.Fatalf("got %q", got)
	}
	if got := normalizeRelationship("nope"); got != "" {
		t.Fatalf("got %q", got)
	}
}
