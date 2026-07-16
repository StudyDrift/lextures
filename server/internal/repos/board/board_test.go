package board

import (
	"strings"
	"testing"

	"github.com/lextures/lextures/server/internal/repos/organization"
)

func TestSuggestSlugFromTitle(t *testing.T) {
	slug := organization.SuggestSlugFromName("Brainstorm Wall!")
	if slug != "brainstorm-wall" {
		t.Fatalf("slug = %q", slug)
	}
}

func TestTitleValidationHelpers(t *testing.T) {
	if maxTitleLen != 200 {
		t.Fatalf("maxTitleLen = %d", maxTitleLen)
	}
	long := strings.Repeat("a", maxTitleLen+1)
	if len(long) <= maxTitleLen {
		t.Fatal("expected long title fixture")
	}
}
