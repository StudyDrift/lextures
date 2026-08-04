package coursechecklist

import (
	"context"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/google/uuid"
)

func TestRegistryIntegrity(t *testing.T) {
	reg, err := BuildBuiltinRegistry()
	if err != nil {
		t.Fatalf("BuildBuiltinRegistry: %v", err)
	}
	if reg.Size() != 101 {
		t.Fatalf("expected 101 rules, got %d", reg.Size())
	}

	routes, err := loadWebRoutesFixture()
	if err != nil {
		t.Fatalf("web routes: %v", err)
	}

	seen := make(map[ItemID]struct{}, reg.Size())
	for _, it := range reg.List() {
		if _, dup := seen[it.ID]; dup {
			t.Errorf("duplicate id %q", it.ID)
		}
		seen[it.ID] = struct{}{}
		if !ItemIDPattern.MatchString(string(it.ID)) {
			t.Errorf("id %q fails regex", it.ID)
		}
		if strings.TrimSpace(it.TitleDefault) == "" || strings.TrimSpace(it.WhyDefault) == "" {
			t.Errorf("id %q missing title/why defaults", it.ID)
		}
		if utf8.RuneCountInString(it.TitleDefault) > 60 {
			t.Errorf("id %q TitleDefault exceeds 60 chars: %q", it.ID, it.TitleDefault)
		}
		lowerTitle := strings.ToLower(it.TitleDefault)
		for _, banned := range []string{"failed", "should have", "!"} {
			if strings.Contains(lowerTitle, banned) {
				t.Errorf("id %q title contains banned %q", it.ID, banned)
			}
		}
		if len(it.Sources) == 0 {
			t.Errorf("id %q missing Sources", it.ID)
		}
		if it.Tier != TierRecommended {
			t.Errorf("id %q tier=%s want recommended (FR-37)", it.ID, it.Tier)
		}
		if err := validateNavTargetRoute(it.Target.Route, routes); err != nil {
			t.Errorf("id %q target: %v", it.ID, err)
		}
		canEmit := it.EvidenceShape != nil
		if canEmit && len(it.EvidenceShape.Columns) == 0 {
			t.Errorf("id %q EvidenceShape missing column headers", it.ID)
		}
		// Smoke: people.sections should emit evidence when students lack sections.
		if canEmit && it.ID == ItemPeopleSections {
			uid := uuid.New()
			f, err := it.Evaluate(context.Background(), CourseSnapshot{
				SectionsEnabled: true,
				Sections:        []SectionSnap{{SectionCode: "A", Name: "Section A"}},
				People: []PersonSnap{
					{UserID: uid, DisplayName: "Stu", Role: "student", Active: true},
				},
			})
			if err != nil {
				t.Fatalf("evaluate %s: %v", it.ID, err)
			}
			if len(f.Evidence) == 0 {
				t.Errorf("id %q has EvidenceShape but emitted no evidence", it.ID)
			}
		}
		if !canEmit && it.ID == ItemCourseDates {
			f, err := it.Evaluate(context.Background(), CourseSnapshot{})
			if err != nil {
				t.Fatalf("evaluate %s: %v", it.ID, err)
			}
			if len(f.Evidence) != 0 {
				t.Errorf("id %q has no EvidenceShape but emitted evidence", it.ID)
			}
		}
	}

	for from, to := range ITEM_ID_ALIASES {
		if _, retired := RETIRED_ITEM_IDS[to]; retired {
			t.Errorf("alias %q → retired %q", from, to)
		}
		if reg.Get(to) == nil {
			t.Errorf("alias %q → missing canonical %q", from, to)
		}
	}

	if _, ok := RETIRED_ITEM_IDS[ItemCourseSections]; !ok {
		t.Fatal("course.sections should be retired")
	}
	if _, ok := reg.ResolveItemID(string(ItemCourseSections)); ok {
		t.Fatal("retired course.sections should not resolve")
	}
}

func TestResolveItemIDAliasAndRetired(t *testing.T) {
	reg := MustDefault()
	id, ok := reg.ResolveItemID(string(ItemCourseDates))
	if !ok || id != ItemCourseDates {
		t.Fatalf("resolve canonical: got %q %v", id, ok)
	}

	ITEM_ID_ALIASES["course.legacy-dates"] = ItemCourseDates
	t.Cleanup(func() { delete(ITEM_ID_ALIASES, "course.legacy-dates") })
	id, ok = reg.ResolveItemID("course.legacy-dates")
	if !ok || id != ItemCourseDates {
		t.Fatalf("alias resolve: got %q %v", id, ok)
	}

	RETIRED_ITEM_IDS["course.gone"] = struct{}{}
	t.Cleanup(func() { delete(RETIRED_ITEM_IDS, "course.gone") })
	if _, ok := reg.ResolveItemID("course.gone"); ok {
		t.Fatal("retired id should not resolve")
	}
}

func TestRulesFilesForbidDatabaseImports(t *testing.T) {
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	fset := token.NewFileSet()
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasPrefix(name, "rules_") || !strings.HasSuffix(name, ".go") {
			continue
		}
		if strings.HasSuffix(name, "_test.go") {
			continue
		}
		src, err := os.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}
		file, err := parser.ParseFile(fset, name, src, parser.ImportsOnly)
		if err != nil {
			t.Fatal(err)
		}
		for _, imp := range file.Imports {
			path := strings.Trim(imp.Path.Value, `"`)
			if strings.Contains(path, "pgx") || strings.Contains(path, "database/sql") ||
				strings.Contains(path, "/repos/") {
				t.Errorf("%s imports database package %q", name, path)
			}
		}
	}
	for _, name := range []string{
		"rules_foundations.go",
		"rules_orientation.go",
		"rules_syllabus.go",
		"rules_people.go",
		"rules_structure.go",
		"rules_outcomes.go",
	} {
		if _, err := os.Stat(filepath.Join(name)); err != nil {
			t.Fatalf("%s missing: %v", name, err)
		}
	}
}
