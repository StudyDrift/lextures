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

func TestLoadCourseSpec_AIEssentials(t *testing.T) {
	spec, err := LoadCourseSpec("ai-essentials")
	if err != nil {
		t.Fatal(err)
	}
	m := spec.Manifest
	if m.Code != "C-AIESS1" {
		t.Fatalf("code: %s", m.Code)
	}
	if m.CatalogSlug != "ai-essentials" {
		t.Fatalf("catalog_slug: %s", m.CatalogSlug)
	}
	if m.CatalogCategory != "Technology" {
		t.Fatalf("catalog_category: %s", m.CatalogCategory)
	}
	if m.DifficultyLevel != "beginner" {
		t.Fatalf("difficulty_level: %s", m.DifficultyLevel)
	}
	if m.PriceCents != 0 {
		t.Fatalf("price_cents: %d", m.PriceCents)
	}
	if !m.MarketplaceListed || !m.IsPublic {
		t.Fatal("expected marketplace_listed and is_public")
	}
	if len(m.Outcomes) != 6 {
		t.Fatalf("outcomes: %d", len(m.Outcomes))
	}
	if len(spec.Modules) != 7 {
		t.Fatalf("modules: %d", len(spec.Modules))
	}
	var promptLab, capstone bool
	for _, mod := range spec.Modules {
		if len(mod.Pages) < 3 {
			t.Fatalf("module %s: want ≥3 pages, got %d", mod.Meta.Slug, len(mod.Pages))
		}
		if len(mod.Quizzes) != 1 {
			t.Fatalf("module %s: want 1 quiz, got %d", mod.Meta.Slug, len(mod.Quizzes))
		}
		q := mod.Quizzes[0]
		n := len(q.Questions)
		if n < 5 || n > 8 {
			t.Fatalf("module %s quiz: want 5–8 questions, got %d", mod.Meta.Slug, n)
		}
		if !q.Grading.UnlimitedAttempts {
			t.Fatalf("module %s quiz: expected unlimited_attempts", mod.Meta.Slug)
		}
		for _, a := range mod.Assignments {
			switch a.Slug {
			case "m5.prompting.prompt-lab":
				promptLab = true
				if a.Grading.Points != 10 || a.Grading.GradePolicy != GradePolicyGraderAgent {
					t.Fatalf("prompt lab grading: %+v", a.Grading)
				}
			case "m7.responsible-ai.capstone":
				capstone = true
				if a.Grading.Points != 15 || a.Grading.GradePolicy != GradePolicyGraderAgent {
					t.Fatalf("capstone grading: %+v", a.Grading)
				}
			}
		}
	}
	if !promptLab {
		t.Fatal("missing Prompt Lab assignment")
	}
	if !capstone {
		t.Fatal("missing capstone assignment")
	}
	if spec.Syllabus == nil || len(spec.Syllabus.Sections) < 5 {
		t.Fatalf("syllabus sections: %+v", spec.Syllabus)
	}
	if err := ValidateCourseSpec(spec); err != nil {
		t.Fatal(err)
	}
}

func TestResolveCourseDir_AIEssentials(t *testing.T) {
	dir, err := ResolveCourseDir("ai-essentials")
	if err != nil || dir != "ai-essentials" {
		t.Fatalf("dir=%s err=%v", dir, err)
	}
	dir, err = ResolveCourseDir("C-AIESS1")
	if err != nil || dir != "ai-essentials" {
		t.Fatalf("by code: dir=%s err=%v", dir, err)
	}
}

func TestIsDeployCourse(t *testing.T) {
	if !IsDeployCourse("ai-essentials") {
		t.Fatal("ai-essentials should provision on deploy")
	}
	if IsDeployCourse(HarnessSmokeDir) {
		t.Fatal("harness-smoke must not provision on deploy")
	}
}

func TestLoadCourseSpec_IntroductionToPython(t *testing.T) {
	spec, err := LoadCourseSpec("introduction-to-python")
	if err != nil {
		t.Fatal(err)
	}
	m := spec.Manifest
	if m.Code != "C-INTPY1" {
		t.Fatalf("code: %s", m.Code)
	}
	if m.CatalogSlug != "introduction-to-python" {
		t.Fatalf("catalog_slug: %s", m.CatalogSlug)
	}
	if m.CatalogCategory != "Programming" {
		t.Fatalf("catalog_category: %s", m.CatalogCategory)
	}
	if m.DifficultyLevel != "beginner" {
		t.Fatalf("difficulty_level: %s", m.DifficultyLevel)
	}
	if m.PriceCents != 0 {
		t.Fatalf("price_cents: %d", m.PriceCents)
	}
	if !m.MarketplaceListed || !m.IsPublic {
		t.Fatal("expected marketplace_listed and is_public")
	}
	if len(m.Outcomes) != 8 {
		t.Fatalf("outcomes: %d", len(m.Outcomes))
	}
	if len(spec.Modules) != 8 {
		t.Fatalf("modules: %d", len(spec.Modules))
	}
	wantAssignments := map[string]int{
		"m4.decisions-looping.even-odd":           10,
		"m5.collections.frequency":                10,
		"m6.functions-modules.temperature":        10,
		"m7.strings-files-errors.file-summarizer": 10,
		"m8.putting-it-together.capstone":         20,
	}
	found := map[string]bool{}
	for _, mod := range spec.Modules {
		if len(mod.Pages) < 3 {
			t.Fatalf("module %s: want ≥3 pages, got %d", mod.Meta.Slug, len(mod.Pages))
		}
		if len(mod.Quizzes) != 1 {
			t.Fatalf("module %s: want 1 quiz, got %d", mod.Meta.Slug, len(mod.Quizzes))
		}
		q := mod.Quizzes[0]
		n := len(q.Questions)
		if n < 5 || n > 8 {
			t.Fatalf("module %s quiz: want 5–8 questions, got %d", mod.Meta.Slug, n)
		}
		if !q.Grading.UnlimitedAttempts {
			t.Fatalf("module %s quiz: expected unlimited_attempts", mod.Meta.Slug)
		}
		hasPredict := false
		for _, question := range q.Questions {
			p := strings.ToLower(question.Prompt)
			if strings.Contains(p, "what does") || strings.Contains(p, "display") || strings.Contains(p, "```") {
				hasPredict = true
				break
			}
		}
		if !hasPredict {
			t.Fatalf("module %s quiz: expected at least one predict-the-output style item", mod.Meta.Slug)
		}
		for _, a := range mod.Assignments {
			pts, ok := wantAssignments[a.Slug]
			if !ok {
				t.Fatalf("unexpected assignment %s", a.Slug)
			}
			found[a.Slug] = true
			if a.Grading.Points != pts || a.Grading.GradePolicy != GradePolicyGraderAgent {
				t.Fatalf("assignment %s grading: %+v", a.Slug, a.Grading)
			}
		}
	}
	for slug := range wantAssignments {
		if !found[slug] {
			t.Fatalf("missing assignment %s", slug)
		}
	}
	if spec.Syllabus == nil || len(spec.Syllabus.Sections) < 5 {
		t.Fatalf("syllabus sections: %+v", spec.Syllabus)
	}
	if err := ValidateCourseSpec(spec); err != nil {
		t.Fatal(err)
	}
}

func TestResolveCourseDir_IntroductionToPython(t *testing.T) {
	dir, err := ResolveCourseDir("introduction-to-python")
	if err != nil || dir != "introduction-to-python" {
		t.Fatalf("dir=%s err=%v", dir, err)
	}
	dir, err = ResolveCourseDir("C-INTPY1")
	if err != nil || dir != "introduction-to-python" {
		t.Fatalf("by code: dir=%s err=%v", dir, err)
	}
}

func TestLoadCourseSpec_PersonalFinance(t *testing.T) {
	spec, err := LoadCourseSpec("personal-finance")
	if err != nil {
		t.Fatal(err)
	}
	m := spec.Manifest
	if m.Code != "C-PERFIN" {
		t.Fatalf("code: %s", m.Code)
	}
	if m.CatalogSlug != "personal-finance" {
		t.Fatalf("catalog_slug: %s", m.CatalogSlug)
	}
	if m.CatalogCategory != "Life Skills" {
		t.Fatalf("catalog_category: %s", m.CatalogCategory)
	}
	if m.DifficultyLevel != "beginner" {
		t.Fatalf("difficulty_level: %s", m.DifficultyLevel)
	}
	if m.PriceCents != 0 {
		t.Fatalf("price_cents: %d", m.PriceCents)
	}
	if !m.MarketplaceListed || !m.IsPublic {
		t.Fatal("expected marketplace_listed and is_public")
	}
	if len(m.Outcomes) != 7 {
		t.Fatalf("outcomes: %d", len(m.Outcomes))
	}
	if len(spec.Modules) != 7 {
		t.Fatalf("modules: %d", len(spec.Modules))
	}
	wantAssignments := map[string]struct {
		points int
		policy string
	}{
		"m2.budgeting-cash-flow.monthly-budget": {
			points: 10,
			policy: GradePolicyCompletionFull,
		},
		"m4.credit-debt.debt-payoff-plan": {
			points: 10,
			policy: GradePolicyCompletionFull,
		},
		"m5.investing-basics.compounding-exercise": {
			points: 10,
			policy: GradePolicyCompletionFull,
		},
		"m7.protecting-money-next-steps.capstone": {
			points: 15,
			policy: GradePolicyGraderAgent,
		},
	}
	found := map[string]bool{}
	disclaimerHit := 0
	for _, mod := range spec.Modules {
		if len(mod.Pages) < 3 {
			t.Fatalf("module %s: want ≥3 pages, got %d", mod.Meta.Slug, len(mod.Pages))
		}
		if len(mod.Quizzes) != 1 {
			t.Fatalf("module %s: want 1 quiz, got %d", mod.Meta.Slug, len(mod.Quizzes))
		}
		q := mod.Quizzes[0]
		n := len(q.Questions)
		if n < 5 || n > 8 {
			t.Fatalf("module %s quiz: want 5–8 questions, got %d", mod.Meta.Slug, n)
		}
		if !q.Grading.UnlimitedAttempts {
			t.Fatalf("module %s quiz: expected unlimited_attempts", mod.Meta.Slug)
		}
		for _, p := range mod.Pages {
			if strings.Contains(p.Markdown, "Educational disclaimer") {
				disclaimerHit++
			}
		}
		for _, a := range mod.Assignments {
			want, ok := wantAssignments[a.Slug]
			if !ok {
				t.Fatalf("unexpected assignment %s", a.Slug)
			}
			found[a.Slug] = true
			if a.Grading.Points != want.points || a.Grading.GradePolicy != want.policy {
				t.Fatalf("assignment %s grading: %+v", a.Slug, a.Grading)
			}
			if !strings.Contains(a.Markdown, "Privacy warning") && !strings.Contains(a.Markdown, "illustrative") {
				t.Fatalf("assignment %s: expected PII-minimizing warning", a.Slug)
			}
		}
	}
	for slug := range wantAssignments {
		if !found[slug] {
			t.Fatalf("missing assignment %s", slug)
		}
	}
	if disclaimerHit < 7 {
		t.Fatalf("expected educational disclaimer on money-decision pages, hits=%d", disclaimerHit)
	}
	if spec.Syllabus == nil || len(spec.Syllabus.Sections) < 5 {
		t.Fatalf("syllabus sections: %+v", spec.Syllabus)
	}
	if err := ValidateCourseSpec(spec); err != nil {
		t.Fatal(err)
	}
}

func TestResolveCourseDir_PersonalFinance(t *testing.T) {
	dir, err := ResolveCourseDir("personal-finance")
	if err != nil || dir != "personal-finance" {
		t.Fatalf("dir=%s err=%v", dir, err)
	}
	dir, err = ResolveCourseDir("C-PERFIN")
	if err != nil || dir != "personal-finance" {
		t.Fatalf("by code: dir=%s err=%v", dir, err)
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
