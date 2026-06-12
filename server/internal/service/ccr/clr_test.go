package ccr

import (
	"testing"
	"time"
)

func TestPortfolioEvidenceURL(t *testing.T) {
	slug := "my-portfolio"
	if got := portfolioEvidenceURL(&slug, true); got != "/portfolios/my-portfolio" {
		t.Fatalf("public slug: got %q", got)
	}
	if got := portfolioEvidenceURL(&slug, false); got != "" {
		t.Fatalf("private artifact: got %q", got)
	}
	if got := portfolioEvidenceURL(nil, true); got != "" {
		t.Fatalf("nil slug: got %q", got)
	}
}

func TestBuildCLRSubjectAssertions(t *testing.T) {
	issued := time.Date(2026, 1, 15, 12, 0, 0, 0, time.UTC)
	achievements := []AggregatedAchievement{
		{
			ID:          "course:abc",
			Type:        "course_completion",
			Title:       "Intro to CS",
			Description: "Final grade: A",
			IssuedAt:    issued,
		},
		{
			ID:          "badge-1",
			Type:        "badge",
			Title:       "Peer Mentor",
			IssuedAt:    issued,
			OutcomeTags: []string{"Leadership"},
		},
	}
	subject := BuildCLRSubject("urn:uuid:user:1", "Alex Student", "did:web:example.com", "Example U", achievements, issued)
	assertions, ok := subject["assertions"].([]map[string]any)
	if !ok {
		t.Fatalf("expected assertions slice, got %T", subject["assertions"])
	}
	if len(assertions) != 2 {
		t.Fatalf("expected 2 assertions, got %d", len(assertions))
	}
}
