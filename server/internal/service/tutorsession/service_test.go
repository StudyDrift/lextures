package tutorsession

import (
	"strings"
	"testing"

	"github.com/google/uuid"
	tutorrepo "github.com/lextures/lextures/server/internal/repos/tutorsession"
)

func TestValidateMessage_RedactsPII(t *testing.T) {
	out, err := ValidateMessage("Contact me at alice@example.com about derivatives")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out, "alice@example.com") {
		t.Fatalf("email not redacted: %q", out)
	}
}

func TestFilterValidCitations_KeepsValidOnly(t *testing.T) {
	retrieved := []tutorrepo.Citation{
		{SourceID: "item-1", ChunkID: "abc", Excerpt: "factoring"},
		{SourceID: "item-2", ChunkID: "def", Excerpt: "FOIL"},
	}
	got := FilterValidCitations([]tutorrepo.Citation{
		{SourceID: "item-1", ChunkID: "abc", Excerpt: "factoring"},
		{SourceID: "fake", ChunkID: "zzz", Excerpt: "bad"},
	}, retrieved)
	if len(got) != 1 || got[0].SourceID != "item-1" {
		t.Fatalf("unexpected citations: %#v", got)
	}
}

func TestFilterValidCitations_FallbackToRetrieved(t *testing.T) {
	retrieved := []tutorrepo.Citation{{SourceID: "item-1", ChunkID: "abc", Excerpt: "x"}}
	got := FilterValidCitations(nil, retrieved)
	if len(got) != 1 || got[0].SourceID != "item-1" {
		t.Fatalf("expected fallback citation: %#v", got)
	}
}

func TestDetectConceptTags(t *testing.T) {
	derivID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	concepts := []ConceptRef{
		{ID: derivID, Name: "Derivatives"},
		{ID: uuid.MustParse("22222222-2222-2222-2222-222222222222"), Name: "Integrals"},
	}
	ids := DetectConceptTags("I'm confused about derivatives again", concepts)
	if len(ids) != 1 || ids[0] != derivID {
		t.Fatalf("unexpected tags: %#v", ids)
	}
}

func TestSessionTitleFromMessage(t *testing.T) {
	long := strings.Repeat("word ", 20)
	title := SessionTitleFromMessage(long)
	if len([]rune(title)) > 48 {
		t.Fatalf("title too long: %q", title)
	}
}

func TestBuildSystemPrompt_IncludesCourseTitle(t *testing.T) {
	prompt := BuildSystemPrompt("Calculus I", true)
	if !strings.Contains(prompt, "Calculus I") {
		t.Fatalf("missing course title: %q", prompt)
	}
}

func TestDisclosureMessage(t *testing.T) {
	if !strings.Contains(DisclosureMessage(), "AI tutor") {
		t.Fatalf("unexpected disclosure: %q", DisclosureMessage())
	}
}
