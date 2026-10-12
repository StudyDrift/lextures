package starterlibrary

import (
	"strings"
	"testing"
)

func TestCatalogCoversCoreSubjects(t *testing.T) {
	cores := []string{SubjectMath, SubjectELA, SubjectScience, SubjectHistory}
	for _, band := range GradeBands {
		for _, subject := range cores {
			got := Filter(band, subject)
			if len(got) < 1 {
				t.Errorf("missing starter for %s %s", band, subject)
			}
		}
	}
	if len(Filter("", SubjectElective)) < 1 {
		t.Fatal("expected at least one elective")
	}
	seen := map[string]struct{}{}
	for _, c := range Catalog() {
		if _, ok := seen[c.Slug]; ok {
			t.Fatalf("duplicate slug %s", c.Slug)
		}
		seen[c.Slug] = struct{}{}
		if c.Title == "" || c.Summary == "" || c.EstimatedMinutes < 20 {
			t.Fatalf("%s missing title, summary, or length", c.Slug)
		}
		if len(c.Modules) < 1 {
			t.Fatalf("%s has no modules", c.Slug)
		}
		for _, mod := range c.Modules {
			if mod.Title == "" || len(mod.Pages) < 1 || mod.Pages[0].Markdown == "" {
				t.Fatalf("%s module %q is incomplete", c.Slug, mod.Title)
			}
		}
	}
}

func TestFilterAndGet(t *testing.T) {
	math := Filter("3-5", SubjectMath)
	if len(math) != 1 || math[0].Slug != "math-3-5" {
		t.Fatalf("filter: %+v", math)
	}
	if _, ok := Get("missing"); ok {
		t.Fatal("expected missing slug")
	}
	got, ok := Get("history-9-12")
	if !ok || !strings.Contains(got.Title, "Modern") {
		t.Fatalf("get: ok=%v title=%q", ok, got.Title)
	}
}

func TestTemplateAndParseOutline(t *testing.T) {
	c, err := TemplateCourse(Band68, SubjectScience, "water cycle at our house")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(c.Summary, "water cycle") {
		t.Fatalf("focus missing from summary: %s", c.Summary)
	}
	if !strings.Contains(c.Modules[0].Pages[0].Markdown, "Your focus") {
		t.Fatal("focus missing from first page")
	}
	if _, err := TemplateCourse("college", SubjectMath, ""); err == nil {
		t.Fatal("expected invalid band")
	}

	raw := `{
	  "title": "Kitchen chemistry",
	  "summary": "Observe mixtures at home.",
	  "estimatedMinutes": 40,
	  "standards": ["MS-PS1-1"],
	  "modules": [
	    {"title": "Mixtures", "pages": [
	      {"title": "Look", "markdown": "Stir salt into warm water and watch it disappear."},
	      {"title": "Try", "markdown": "Compare salt water with sand in water."}
	    ]}
	  ]
	}`
	parsed, err := ParseOutline(raw, Band68, SubjectScience)
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Title != "Kitchen chemistry" || parsed.GradeBand != Band68 || len(parsed.Modules) != 1 {
		t.Fatalf("parsed: %+v", parsed)
	}
	if _, err := ParseOutline(`{"title":""}`, Band68, SubjectScience); err == nil {
		t.Fatal("expected parse failure")
	}
}
