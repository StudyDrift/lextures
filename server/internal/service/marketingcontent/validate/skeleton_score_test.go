package validate

import (
	"strings"
	"testing"
)

func TestInsertBlockSkeletonsDoNotEmitRawHTML(t *testing.T) {
	t.Parallel()
	// Mirrors clients/web article-editor-utils directive skeletons (no HTML comments).
	body := ":::key-takeaways\n- First takeaway\n- Second takeaway\n- Third takeaway\n:::\n\n:::answer\n" +
		strings.Repeat("complete answer words ", 15) +
		"\n:::\n\n## How does this work?\n\n" +
		strings.Repeat("clear passage words ", 45) +
		" [one](/one) [two](/two) [three](/three) [^1]\n\n:::faq\n### What is one?\nA.\n### What is two?\nB.\n### What is three?\nC.\n:::\n\n[^1]: https://example.com"
	report := Article(Input{Kind: "blog", BodyMD: body, Metadata: validMetadata(), KnownPaths: map[string]struct{}{"/one": {}, "/two": {}, "/three": {}}})
	if report.Score < 8 {
		t.Fatalf("expected publishable score, got %.1f findings=%+v", report.Score, report.Findings)
	}
	for _, f := range report.Findings {
		if f.Severity == "error" {
			t.Fatalf("unexpected blocking finding: %+v", f)
		}
	}
}

func TestTipTapSerializedCitationsEarnScore(t *testing.T) {
	t.Parallel()
	// TipTap markdown round-trip turns [^1] + [^1]: url into [^1](url).
	body := ":::key-takeaways\n- One conclusion\n- Two conclusion\n- Three conclusion\n:::\n\n:::answer\n" +
		strings.Repeat("complete answer words ", 15) +
		"\n:::\n\n## How does this work?\n\n" +
		strings.Repeat("clear passage words ", 45) +
		" About 42% teachers benefit [^1](https://example.com). [one](/one) [two](/two) [three](/three)\n\n:::faq\n### What is one?\nA.\n### What is two?\nB.\n### What is three?\nC.\n:::\n"
	report := Article(Input{Kind: "blog", BodyMD: body, Metadata: validMetadata(), KnownPaths: map[string]struct{}{"/one": {}, "/two": {}, "/three": {}}})
	if report.Score < 8 {
		t.Fatalf("TipTap citation form should earn citation points; score=%.1f findings=%+v", report.Score, report.Findings)
	}
	for _, f := range report.Findings {
		if f.Rule == "cite.source-resolvable" || f.Rule == "cite.numeric-claim" {
			t.Fatalf("unexpected citation finding for TipTap form: %+v", f)
		}
	}
}

func TestPassageStatsIgnoreDirectiveBlocks(t *testing.T) {
	t.Parallel()
	shortPassage := strings.Repeat("clear passage words ", 30) // 90 words — below 120
	body := ":::key-takeaways\n- One conclusion about learning\n- Two conclusion about teaching\n- Three conclusion about schools\n:::\n\n:::answer\n" +
		strings.Repeat("complete answer words ", 15) +
		"\n:::\n\n## How does this work?\n\n" + shortPassage + "\n\n:::faq\n### What is one?\nA.\n### What is two?\nB.\n### What is three?\nC.\n:::\n"
	report := Article(Input{Kind: "blog", BodyMD: body, Metadata: validMetadata(), KnownPaths: map[string]struct{}{}})
	// If answer/takeaways were counted as passages, mean could wrongly enter 120–180.
	if !has(report, "passage.self-contained", "warn") {
		t.Fatalf("expected short prose passage warning when directives are excluded; score=%.1f findings=%+v", report.Score, report.Findings)
	}
}
