package coursechecklist

import (
	"context"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
)

func TestStructureCopyContract(t *testing.T) {
	for _, it := range structureRules() {
		if utf8.RuneCountInString(it.TitleDefault) > 60 {
			t.Errorf("%s title too long", it.ID)
		}
		low := strings.ToLower(it.TitleDefault + " " + it.WhyDefault)
		for _, ban := range []string{"failed", "should have", "!"} {
			if strings.Contains(low, ban) {
				t.Errorf("%s contains banned %q", it.ID, ban)
			}
		}
		if it.EvidenceShape != nil && len(it.EvidenceShape.Columns) == 0 {
			t.Errorf("%s EvidenceShape missing columns", it.ID)
		}
	}
}

func TestStructureEmptyModulesAC7(t *testing.T) {
	// AC-7
	m1, m2 := uuid.New(), uuid.New()
	f, err := findRule(t, ItemStructureEmptyModules).Evaluate(context.Background(), CourseSnapshot{
		StructureItems: []StructureItem{
			{ID: m1, Kind: "module", Title: "Week 1", SortOrder: 0},
			{ID: m2, Kind: "module", Title: "Week 2", SortOrder: 1},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if f.Status != StatusTodo || len(f.Evidence) != 2 {
		t.Fatalf("got status=%s evidence=%d detail=%q", f.Status, len(f.Evidence), f.DetailDefault)
	}
}

func TestStructureUnpublishedItemsAC8(t *testing.T) {
	// AC-8
	mod := uuid.New()
	past := time.Now().UTC().Add(-24 * time.Hour)
	items := []StructureItem{
		{ID: mod, Kind: "module", Title: "M", Published: true, VisibleFrom: &past, SortOrder: 0},
		{ID: uuid.New(), Kind: "content_page", Title: "A", ParentID: &mod, Published: false, SortOrder: 0},
		{ID: uuid.New(), Kind: "content_page", Title: "B", ParentID: &mod, Published: false, SortOrder: 1},
		{ID: uuid.New(), Kind: "assignment", Title: "C", ParentID: &mod, Published: false, SortOrder: 2},
	}
	rule := findRule(t, ItemStructureUnpublishedItems)
	f, _ := rule.Evaluate(context.Background(), CourseSnapshot{Published: true, StructureItems: items})
	if f.Status != StatusTodo || len(f.Evidence) != 3 {
		t.Fatalf("published course: %+v", f)
	}
	f, _ = rule.Evaluate(context.Background(), CourseSnapshot{Published: false, StructureItems: items})
	if f.Status != StatusNotApplicable {
		t.Fatalf("unpublished course: %s", f.Status)
	}
}

func TestStructureGatingReviewAC9(t *testing.T) {
	// AC-9 — cycle A→B→A terminates with todo
	a, b := uuid.New(), uuid.New()
	f, err := findRule(t, ItemStructureGatingReview).Evaluate(context.Background(), CourseSnapshot{
		ModuleGatingEnabled: true,
		StructureItems: []StructureItem{
			{ID: a, Kind: "module", Title: "A", Published: true},
			{ID: b, Kind: "module", Title: "B", Published: true},
		},
		ModulePrerequisiteEdges: []PrerequisiteEdge{
			{ModuleID: a, PrerequisiteModuleID: b},
			{ModuleID: b, PrerequisiteModuleID: a},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if f.Status != StatusTodo {
		t.Fatalf("status=%s", f.Status)
	}
	foundCycle := false
	for _, e := range f.Evidence {
		if e.Sublabel == "cycle" {
			foundCycle = true
		}
	}
	if !foundCycle {
		t.Fatalf("expected cycle evidence, got %+v", f.Evidence)
	}
}

func TestStructurePlaceholderTitles(t *testing.T) {
	mod := uuid.New()
	f, _ := findRule(t, ItemStructurePlaceholderTitles).Evaluate(context.Background(), CourseSnapshot{
		CatalogLanguage: "en",
		StructureItems: []StructureItem{
			{ID: mod, Kind: "module", Title: "M"},
			{ID: uuid.New(), Kind: "content_page", Title: "Untitled", ParentID: &mod, SortOrder: 0},
			{ID: uuid.New(), Kind: "content_page", Title: "Real title", ParentID: &mod, SortOrder: 1},
			{ID: uuid.New(), Kind: "assignment", Title: "New assignment draft", ParentID: &mod, SortOrder: 2},
		},
	})
	if f.Status != StatusTodo || len(f.Evidence) != 2 {
		t.Fatalf("got %+v", f)
	}
}

func TestStructureModulesExistAndPacing(t *testing.T) {
	f, _ := findRule(t, ItemStructureModulesExist).Evaluate(context.Background(), CourseSnapshot{})
	if f.Status != StatusTodo {
		t.Fatalf("empty=%s", f.Status)
	}
	mod := uuid.New()
	due := time.Now().UTC()
	f, _ = findRule(t, ItemStructureModulesExist).Evaluate(context.Background(), CourseSnapshot{
		StructureItems: []StructureItem{{ID: mod, Kind: "module", Title: "M"}},
	})
	if f.Status != StatusDone {
		t.Fatalf("with module=%s", f.Status)
	}

	f, _ = findRule(t, ItemStructurePacingSignal).Evaluate(context.Background(), CourseSnapshot{
		StructureItems: []StructureItem{
			{ID: mod, Kind: "module", Title: "M"},
			{ID: uuid.New(), Kind: "assignment", Title: "A", ParentID: &mod},
		},
	})
	if f.Status != StatusTodo {
		t.Fatalf("no due=%s", f.Status)
	}
	f, _ = findRule(t, ItemStructurePacingSignal).Evaluate(context.Background(), CourseSnapshot{
		StructureItems: []StructureItem{
			{ID: mod, Kind: "module", Title: "M"},
			{ID: uuid.New(), Kind: "assignment", Title: "A", ParentID: &mod, DueAt: &due},
		},
	})
	if f.Status != StatusDone {
		t.Fatalf("with due=%s", f.Status)
	}
}

func TestFindPrerequisiteCyclesDepthCap(t *testing.T) {
	// Adversarial graph size — must terminate.
	adj := map[uuid.UUID][]uuid.UUID{}
	nodes := make([]uuid.UUID, 500)
	for i := range nodes {
		nodes[i] = uuid.New()
	}
	for i := 0; i < len(nodes)-1; i++ {
		adj[nodes[i]] = []uuid.UUID{nodes[i+1]}
	}
	adj[nodes[len(nodes)-1]] = []uuid.UUID{nodes[0]} // one big cycle
	cycles := findPrerequisiteCycles(adj)
	if len(cycles) == 0 {
		t.Fatal("expected a cycle")
	}
}
