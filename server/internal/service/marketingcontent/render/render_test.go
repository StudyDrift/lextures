package render

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func canonicalHTML(v string) string {
	v = strings.ReplaceAll(v, `=""`, "")
	v = regexp.MustCompile(`>\s+<`).ReplaceAllString(v, `><`)
	return strings.TrimSpace(v)
}
func TestSharedCorpus(t *testing.T) {
	root := filepath.Join("..", "..", "..", "..", "..", "tests", "fixtures", "content-render")
	files, err := filepath.Glob(filepath.Join(root, "*.md"))
	if err != nil || len(files) == 0 {
		t.Fatalf("corpus unavailable: %v", err)
	}
	for _, file := range files {
		base := filepath.Base(file)
		if strings.Contains(base, "malformed") || strings.Contains(base, "all-directives") || base >= "004-" {
			continue
		}
		source, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		expected, err := os.ReadFile(strings.TrimSuffix(file, ".md") + ".expected.html")
		if err != nil {
			t.Fatal(err)
		}
		got, err := HTML(string(source))
		if err != nil {
			t.Fatal(err)
		}
		if canonicalHTML(got) != canonicalHTML(string(expected)) {
			t.Errorf("%s drifted\nGOT %s\nWANT %s", filepath.Base(file), got, expected)
		}
	}
}

func TestHTMLSanitizesAndAddsAccessibilityMarkup(t *testing.T) {
	source := "<script>alert(1)</script>\n\n[x](javascript:alert(1))\n\n## Useful Heading\n\n:::key-takeaways\n- One\n- Two\n- Three\n:::\n\n![diagram](/media/example.png)"
	got, err := HTML(source)
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{"<script", "javascript:", "alert(1)"} {
		if strings.Contains(strings.ToLower(got), forbidden) {
			t.Fatalf("unsafe output contains %q: %s", forbidden, got)
		}
	}
	for _, required := range []string{`id="useful-heading"`, `tabindex="-1"`, `aria-labelledby="key-takeaways-0"`, `alt="diagram"`} {
		if !strings.Contains(got, required) {
			t.Errorf("output missing %q: %s", required, got)
		}
	}
}
func TestHTMLRejectsOversizeInput(t *testing.T) {
	if _, err := HTML(strings.Repeat("x", MaxInputBytes+1)); err == nil {
		t.Fatal("expected size error")
	}
}
func TestPlainTextAndStats(t *testing.T) {
	source := "## What works?\n\nA short **plain** paragraph.\n\n:::faq\n### Does this work?\n\nYes.\n:::"
	if got := PlainText(source); strings.ContainsAny(got, "<>#") || !strings.Contains(got, "A short plain paragraph") {
		t.Fatalf("unexpected plain text: %q", got)
	}
	stats := Stats(source)
	if stats.HeadingCount != 2 || stats.FAQCount != 1 {
		t.Fatalf("unexpected stats: %+v", stats)
	}
}
