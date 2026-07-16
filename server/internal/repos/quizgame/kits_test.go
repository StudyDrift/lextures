package quizgame

import (
	"strings"
	"testing"

	"github.com/lextures/lextures/server/internal/repos/organization"
)

func TestSuggestSlugFromTitle(t *testing.T) {
	got := organization.SuggestSlugFromName("Week 1 Live Quiz!")
	if got == "" {
		t.Fatal("expected non-empty slug")
	}
	if strings.Contains(got, " ") {
		t.Fatalf("slug should be URL-safe, got %q", got)
	}
}

func TestTitleMaxLen(t *testing.T) {
	if maxTitleLen != 200 {
		t.Fatalf("maxTitleLen=%d want 200", maxTitleLen)
	}
	long := strings.Repeat("a", maxTitleLen+1)
	if len(long) <= maxTitleLen {
		t.Fatal("fixture should exceed maxTitleLen")
	}
}

func TestNormalizePage(t *testing.T) {
	p, ps := normalizePage(0, 0)
	if p != 1 || ps != defaultPageSize {
		t.Fatalf("got page=%d pageSize=%d", p, ps)
	}
	_, ps = normalizePage(1, 500)
	if ps != maxPageSize {
		t.Fatalf("expected max page size %d, got %d", maxPageSize, ps)
	}
}
