package validate

import (
	"strings"
	"testing"
)

func validMetadata() Metadata {
	return Metadata{Title: "Title", Description: strings.Repeat("description ", 12), Updated: "2026-08-12", Author: "author", Cluster: "assessment", PrimaryQuestion: "What works?", Keywords: []string{"assessment"}}
}
func has(report Report, rule, severity string) bool {
	for _, f := range report.Findings {
		if f.Rule == rule && f.Severity == severity {
			return true
		}
	}
	return false
}
func TestArticleReportsContractAndSafetyErrors(t *testing.T) {
	report := Article(Input{Kind: "blog", BodyMD: "<script>x</script>\n[x](javascript:alert(1))\n[missing](/docs/nope)\n![](/image.png)\n:::wat\nx\n:::", Metadata: validMetadata(), KnownPaths: map[string]struct{}{}})
	for _, tc := range []struct{ rule, sev string }{{"safety.raw-html", "error"}, {"safety.script-url", "error"}, {"directive.unknown", "error"}, {"link.internal-resolves", "error"}, {"a11y.image-alt", "error"}, {"struct.answer-block", "error"}} {
		if !has(report, tc.rule, tc.sev) {
			t.Errorf("missing %s/%s in %+v", tc.rule, tc.sev, report.Findings)
		}
	}
}
func TestArticleAllowsKnownInternalPathAndLocatesFinding(t *testing.T) {
	report := Article(Input{Kind: "blog", BodyMD: "first\n[known](/docs/known)\n[bad](/docs/bad)", Metadata: validMetadata(), KnownPaths: map[string]struct{}{`/docs/known`: {}}})
	count := 0
	for _, f := range report.Findings {
		if f.Rule == "link.internal-resolves" {
			count++
			if f.Line != 3 || f.Column < 1 {
				t.Fatalf("bad location: %+v", f)
			}
		}
	}
	if count != 1 {
		t.Fatalf("wanted one link finding, got %d", count)
	}
}
func TestScoreMatchesPublishedFormula(t *testing.T) {
	answer := strings.Repeat("complete answer words ", 15)
	passage := strings.Repeat("clear passage words ", 45)
	body := ":::key-takeaways\n- One conclusion\n- Two conclusion\n- Three conclusion\n:::\n\n:::answer\n" + answer + "\n:::\n\n## How does this work?\n\n" + passage + " [one](/one) [two](/two) [three](/three) [^1]\n\n:::faq\n### What is one?\nA.\n### What is two?\nB.\n### What is three?\nC.\n:::\n\n[^1]: https://example.com"
	report := Article(Input{Kind: "blog", BodyMD: body, Metadata: validMetadata(), KnownPaths: map[string]struct{}{`/one`: {}, `/two`: {}, `/three`: {}}})
	if report.Score < 8 {
		t.Fatalf("expected publishable score, got %.1f: %+v", report.Score, report.Findings)
	}
}
