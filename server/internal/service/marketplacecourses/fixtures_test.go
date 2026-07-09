package marketplacecourses

import (
	"strings"
	"testing"
)

func TestLoadCourseSpec_HarnessSmoke(t *testing.T) {
	spec, err := LoadCourseSpec("harness-smoke")
	if err != nil {
		t.Fatal(err)
	}
	if spec.Manifest.Code != "C-MCSMOK" {
		t.Fatalf("code: %s", spec.Manifest.Code)
	}
	if spec.Manifest.PriceCents != 0 {
		t.Fatalf("price_cents: %d", spec.Manifest.PriceCents)
	}
	if !spec.Manifest.MarketplaceListed {
		t.Fatal("expected marketplace_listed")
	}
	if len(spec.Modules) != 1 {
		t.Fatalf("modules: %d", len(spec.Modules))
	}
	if len(spec.Modules[0].Pages) < 1 {
		t.Fatal("expected at least one page")
	}
	if len(spec.Modules[0].Quizzes) != 1 {
		t.Fatalf("quizzes: %d", len(spec.Modules[0].Quizzes))
	}
	if err := ValidateCourseSpec(spec); err != nil {
		t.Fatal(err)
	}
}

func TestValidateAllCourses(t *testing.T) {
	if err := ValidateAllCourses(); err != nil {
		t.Fatal(err)
	}
}

func TestValidateCourseSpec_RejectsNonZeroPrice(t *testing.T) {
	spec, err := LoadCourseSpec("harness-smoke")
	if err != nil {
		t.Fatal(err)
	}
	spec.Manifest.PriceCents = 999
	err = ValidateCourseSpec(spec)
	if err == nil {
		t.Fatal("expected price validation error")
	}
	if !strings.Contains(err.Error(), "price_cents") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateCourseSpec_RejectsQuizWithoutCorrectAnswer(t *testing.T) {
	spec, err := LoadCourseSpec("harness-smoke")
	if err != nil {
		t.Fatal(err)
	}
	q := &spec.Modules[0].Quizzes[0]
	q.Questions[0].CorrectChoiceIndex = nil
	q.FilePath = "content/harness-smoke/en/m1-welcome/knowledge-check.json"
	err = ValidateCourseSpec(spec)
	if err == nil {
		t.Fatal("expected quiz validation error")
	}
	if !strings.Contains(err.Error(), "correctChoiceIndex") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateCourseSpec_RejectsMissingAltText(t *testing.T) {
	spec, err := LoadCourseSpec("harness-smoke")
	if err != nil {
		t.Fatal(err)
	}
	spec.Modules[0].Pages[0].Markdown += "\n\n![](diagram.png)\n"
	err = ValidateCourseSpec(spec)
	if err == nil {
		t.Fatal("expected alt text validation error")
	}
	if !strings.Contains(err.Error(), "alt text") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateCourseSpec_RejectsClickHere(t *testing.T) {
	spec, err := LoadCourseSpec("harness-smoke")
	if err != nil {
		t.Fatal(err)
	}
	spec.Modules[0].Pages[0].Markdown += "\n\n[click here](/courses)\n"
	err = ValidateCourseSpec(spec)
	if err == nil {
		t.Fatal("expected descriptive link validation error")
	}
}

func TestParseCourseYAML_Outcomes(t *testing.T) {
	m, err := parseCourseYAML(`code: C-ABCDEF
title: T
catalog_slug: s
catalog_category: Cat
difficulty_level: beginner
summary: Sum
outcomes:
  - One
  - Two
price_cents: 0
marketplace_listed: true
content_version: 2
`)
	if err != nil {
		t.Fatal(err)
	}
	if len(m.Outcomes) != 2 || m.Outcomes[0] != "One" || m.ContentVersion != 2 {
		t.Fatalf("%+v", m)
	}
}

func TestResolveCourseDir(t *testing.T) {
	dir, err := ResolveCourseDir("harness-smoke")
	if err != nil || dir != "harness-smoke" {
		t.Fatalf("dir=%s err=%v", dir, err)
	}
	dir, err = ResolveCourseDir("C-MCSMOK")
	if err != nil || dir != "harness-smoke" {
		t.Fatalf("by code: dir=%s err=%v", dir, err)
	}
}

func TestSplitFrontMatter(t *testing.T) {
	fm, body, err := splitFrontMatter("---\nslug: test\n---\n\nHello")
	if err != nil {
		t.Fatal(err)
	}
	if fm["slug"] != "test" || body != "Hello" {
		t.Fatalf("fm=%v body=%q", fm, body)
	}
}

func TestAllDesiredSlugs(t *testing.T) {
	spec, err := LoadCourseSpec("harness-smoke")
	if err != nil {
		t.Fatal(err)
	}
	slugs := AllDesiredSlugs(spec)
	if len(slugs) < 3 {
		t.Fatalf("expected module+page+quiz(+assignment), got %v", slugs)
	}
}
