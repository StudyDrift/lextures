package validate

import (
	"encoding/json"
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
func TestArticleAllowsStaticHubsWithoutCatalog(t *testing.T) {
	report := Article(Input{
		Kind:     "blog",
		BodyMD:   "See the [blog](/blog), [docs](/docs/), and [platform](/platform#overview).",
		Metadata: validMetadata(),
	})
	for _, f := range report.Findings {
		if f.Rule == "link.internal-resolves" {
			t.Fatalf("static hub should resolve: %+v", f)
		}
	}
}

func TestArticleStillRejectsUnknownDeepLinks(t *testing.T) {
	report := Article(Input{
		Kind:       "blog",
		BodyMD:     "See the [blog](/blog) and a [missing doc](/docs/nope).",
		Metadata:   validMetadata(),
		KnownPaths: map[string]struct{}{},
	})
	count := 0
	for _, f := range report.Findings {
		if f.Rule == "link.internal-resolves" {
			count++
			if !strings.Contains(f.Message, `/docs/nope`) {
				t.Fatalf("expected unknown deep link, got %+v", f)
			}
		}
	}
	if count != 1 {
		t.Fatalf("wanted one unknown deep link, got %d: %+v", count, report.Findings)
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
func TestFindingJSONUsesCamelCase(t *testing.T) {
	raw, err := json.Marshal(Finding{Rule: "fm.cluster", Severity: "error", Message: "Required metadata field is missing.", Path: "cluster", Line: 0, Column: 0})
	if err != nil {
		t.Fatal(err)
	}
	var keys map[string]any
	if err := json.Unmarshal(raw, &keys); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"rule", "severity", "message", "path"} {
		if _, ok := keys[key]; !ok {
			t.Errorf("missing %q in %s", key, raw)
		}
	}
	for _, key := range []string{"Rule", "Severity", "Message"} {
		if _, ok := keys[key]; ok {
			t.Errorf("PascalCase %q leaked in %s", key, raw)
		}
	}
}

func TestExtractabilityScoreIsAdvisory(t *testing.T) {
	report := Article(Input{Kind: "blog", BodyMD: "short draft", Metadata: validMetadata()})
	if report.Score >= 8 {
		t.Fatalf("expected a score below the floor, got %.1f", report.Score)
	}
	if !has(report, "extractability.score", "warn") {
		t.Fatalf("expected extractability to be a suggestion, got %+v", report.Findings)
	}
	if has(report, "extractability.score", "error") {
		t.Fatalf("extractability must not block publish: %+v", report.Findings)
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
