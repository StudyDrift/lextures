package adminsearch

import (
	"strings"
	"testing"
)

func TestParseTypes_DefaultAll(t *testing.T) {
	f := ParseTypes("")
	if !f.Users || !f.Courses || !f.Content {
		t.Fatalf("expected all types, got %+v", f)
	}
}

func TestParseTypes_SingleType(t *testing.T) {
	f := ParseTypes("courses")
	if f.Users || f.Content || !f.Courses {
		t.Fatalf("got %+v", f)
	}
}

func TestParseTypes_UnknownFallsBackToAll(t *testing.T) {
	f := ParseTypes("unknown")
	if !f.Users || !f.Courses || !f.Content {
		t.Fatalf("got %+v", f)
	}
}

func TestScrubQueryPII_NoEmail(t *testing.T) {
	got := ScrubQueryPII("johnson")
	if got != "johnson" {
		t.Fatalf("got %q", got)
	}
}

func TestScrubQueryPII_EmailHashed(t *testing.T) {
	got := ScrubQueryPII("find alice@school.edu")
	if strings.Contains(got, "alice@school.edu") {
		t.Fatal("expected email to be scrubbed")
	}
	if !strings.HasPrefix(got, "find email:") {
		t.Fatalf("expected email hash prefix, got %q", got)
	}
}

func TestScrubQueryPII_ConsistentHash(t *testing.T) {
	a := ScrubQueryPII("alice@school.edu")
	b := ScrubQueryPII("alice@school.edu")
	if a != b {
		t.Fatalf("hash not stable: %q vs %q", a, b)
	}
}

func TestTotalPages(t *testing.T) {
	if totalPages(0, 25) != 0 {
		t.Fatal("zero total")
	}
	if totalPages(26, 25) != 2 {
		t.Fatal("expected 2 pages")
	}
}
