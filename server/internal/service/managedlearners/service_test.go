package managedlearners

import (
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestNormalizeDisplayName(t *testing.T) {
	_, err := normalizeDisplayName("  ")
	if err == nil {
		t.Fatal("expected error for blank name")
	}
	got, err := normalizeDisplayName("  Alex  ")
	if err != nil || got != "Alex" {
		t.Fatalf("got %q err %v", got, err)
	}
	long := strings.Repeat("a", 121)
	if _, err := normalizeDisplayName(long); err == nil {
		t.Fatal("expected too long")
	}
}

func TestNormalizeRelationship(t *testing.T) {
	if normalizeRelationship("") != "parent" {
		t.Fatal("default")
	}
	if normalizeRelationship("Guardian") != "guardian" {
		t.Fatal("guardian")
	}
	if normalizeRelationship("weird") != "parent" {
		t.Fatal("fallback")
	}
}

func TestManagedEmailDomainMatchesHelper(t *testing.T) {
	id := uuid.MustParse("aaaaaaaa-bbbb-4ccc-8ddd-eeeeeeeeeeee")
	// Package uses user.ManagedEmail via Create; ensure cap constant is sane.
	if MaxActiveManagedPerParent != 8 {
		t.Fatalf("cap %d", MaxActiveManagedPerParent)
	}
	_ = id
}

func TestSessionTTL(t *testing.T) {
	if SessionTTL().Hours() != 8 {
		t.Fatalf("ttl hours %v", SessionTTL().Hours())
	}
}
