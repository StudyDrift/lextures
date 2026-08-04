package coursechecklist

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/google/uuid"
)

func TestOutcomesCopyContract(t *testing.T) {
	for _, it := range outcomesRules() {
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

func TestOutcomesAssessmentMappingAC1(t *testing.T) {
	// AC-1: 24 gradable, 13 mapped → in_progress progress={13,24}, 11 evidence rows
	mod := uuid.New()
	var items []StructureItem
	items = append(items, StructureItem{ID: mod, Kind: "module", Title: "Unit", SortOrder: 0})
	meta := map[uuid.UUID]ItemMeta{}
	var gradableIDs []uuid.UUID
	for i := 0; i < 24; i++ {
		id := uuid.New()
		gradableIDs = append(gradableIDs, id)
		kind := "assignment"
		if i%2 == 1 {
			kind = "quiz"
		}
		pts := 10
		items = append(items, StructureItem{
			ID: id, Kind: kind, Title: "Item " + id.String()[:8],
			ParentID: &mod, SortOrder: i,
		})
		meta[id] = ItemMeta{Kind: kind, PointsWorth: &pts}
	}
	var links []OutcomeLinkSnap
	oid := uuid.New()
	for i := 0; i < 13; i++ {
		links = append(links, OutcomeLinkSnap{OutcomeID: oid, ItemID: gradableIDs[i]})
	}
	snap := CourseSnapshot{
		StructureItems:   items,
		ItemMeta:         meta,
		OutcomeLinks:     links,
		GradableItems:    computeGradableItems(CourseSnapshot{StructureItems: items, ItemMeta: meta}),
		GradableComputed: true,
	}
	f, err := findRule(t, ItemOutcomesAssessmentMapping).Evaluate(context.Background(), snap)
	if err != nil {
		t.Fatal(err)
	}
	if f.Status != StatusInProgress {
		t.Fatalf("status=%s", f.Status)
	}
	if f.Progress == nil || f.Progress.Done != 13 || f.Progress.Total != 24 {
		t.Fatalf("progress=%+v", f.Progress)
	}
	if len(f.Evidence) != 11 {
		t.Fatalf("evidence=%d want 11", len(f.Evidence))
	}
	// Ordered by module then item sort order — first unmapped is index 13.
	if !strings.Contains(f.Evidence[0].Label, gradableIDs[13].String()[:8]) {
		t.Fatalf("first evidence label=%q want item 13", f.Evidence[0].Label)
	}
}

func TestOutcomesAssessmentMappingAC3Done(t *testing.T) {
	// AC-3
	mod := uuid.New()
	a1, a2 := uuid.New(), uuid.New()
	pts := 5
	snap := CourseSnapshot{
		StructureItems: []StructureItem{
			{ID: mod, Kind: "module", Title: "M"},
			{ID: a1, Kind: "assignment", Title: "A1", ParentID: &mod, SortOrder: 0},
			{ID: a2, Kind: "quiz", Title: "Q1", ParentID: &mod, SortOrder: 1},
		},
		ItemMeta: map[uuid.UUID]ItemMeta{
			a1: {Kind: "assignment", PointsWorth: &pts},
			a2: {Kind: "quiz", PointsWorth: &pts},
		},
		OutcomeLinks: []OutcomeLinkSnap{
			{OutcomeID: uuid.New(), ItemID: a1},
			{OutcomeID: uuid.New(), ItemID: a2},
		},
	}
	snap.GradableItems = computeGradableItems(snap)
	snap.GradableComputed = true
	f, _ := findRule(t, ItemOutcomesAssessmentMapping).Evaluate(context.Background(), snap)
	if f.Status != StatusDone {
		t.Fatalf("status=%s", f.Status)
	}
}

func TestOutcomesCoverageAC4(t *testing.T) {
	// AC-4
	o1, o2 := uuid.New(), uuid.New()
	f, _ := findRule(t, ItemOutcomesCoverage).Evaluate(context.Background(), CourseSnapshot{
		Outcomes: []OutcomeSnap{
			{ID: o1, Title: "Explain X", SortOrder: 0},
			{ID: o2, Title: "Apply Y", SortOrder: 1},
		},
		OutcomeLinks: []OutcomeLinkSnap{
			{OutcomeID: o1, ItemID: uuid.New()},
		},
	})
	if f.Status != StatusTodo || len(f.Evidence) != 1 || f.Evidence[0].Label != "Apply Y" {
		t.Fatalf("got %+v", f)
	}
}

func TestOutcomesSummativeVsCoverageAC5(t *testing.T) {
	// AC-5
	o := uuid.New()
	snap := CourseSnapshot{
		Outcomes: []OutcomeSnap{{ID: o, Title: "Analyze data"}},
		OutcomeLinks: []OutcomeLinkSnap{
			{OutcomeID: o, ItemID: uuid.New(), MeasurementLevel: "formative"},
		},
	}
	cov, _ := findRule(t, ItemOutcomesCoverage).Evaluate(context.Background(), snap)
	if cov.Status != StatusDone {
		t.Fatalf("coverage should treat formative as covered: %s", cov.Status)
	}
	sum, _ := findRule(t, ItemOutcomesSummativeCoverage).Evaluate(context.Background(), snap)
	if sum.Status != StatusTodo || len(sum.Evidence) != 1 {
		t.Fatalf("summative=%+v", sum)
	}
}

func TestOutcomesMeasurableAC6(t *testing.T) {
	// AC-6
	rule := findRule(t, ItemOutcomesMeasurable)
	f, _ := rule.Evaluate(context.Background(), CourseSnapshot{
		CatalogLanguage: "en",
		Outcomes: []OutcomeSnap{
			{ID: uuid.New(), Title: "Understand recursion"},
			{ID: uuid.New(), Title: "Implement a recursive solution"},
		},
	})
	if f.Status != StatusTodo || len(f.Evidence) != 1 {
		t.Fatalf("got %+v", f)
	}
	if f.Evidence[0].Label != "Understand recursion" {
		t.Fatalf("label=%q", f.Evidence[0].Label)
	}
	if !strings.Contains(f.Evidence[0].Sublabel, "explain") {
		t.Fatalf("suggestion=%q", f.Evidence[0].Sublabel)
	}
}

func TestOutcomesStandardsAlignmentAC10(t *testing.T) {
	// AC-10
	rule := findRule(t, ItemOutcomesStandardsAlignment)
	if rule.Applies == nil {
		t.Fatal("Applies required")
	}
	snap := CourseSnapshot{Features: CourseFeatures{StandardsAlignmentEnabled: false}}
	if rule.Applies(snap) {
		t.Fatal("should not apply when standards off")
	}
	// Direct evaluate still returns N/A when flag false.
	f, _ := rule.Evaluate(context.Background(), snap)
	if f.Status != StatusNotApplicable {
		t.Fatalf("status=%s", f.Status)
	}
}

func TestSharedGradableSetAC13(t *testing.T) {
	// AC-13 — GradableItemsFor reuses the precomputed set.
	mod := uuid.New()
	a := uuid.New()
	pts := 10
	snap := CourseSnapshot{
		StructureItems: []StructureItem{
			{ID: mod, Kind: "module", Title: "M"},
			{ID: a, Kind: "assignment", Title: "A", ParentID: &mod},
			{ID: uuid.New(), Kind: "survey", Title: "S", ParentID: &mod},
		},
		ItemMeta: map[uuid.UUID]ItemMeta{
			a: {Kind: "assignment", PointsWorth: &pts},
		},
	}
	first := computeGradableItems(snap)
	snap.GradableItems = first
	snap.GradableComputed = true
	second := GradableItemsFor(snap)
	if len(second) != 1 || second[0].ID != a {
		t.Fatalf("gradable=%+v", second)
	}
	// Same backing slice identity when precomputed.
	if &second[0] != &snap.GradableItems[0] {
		t.Fatal("expected shared gradable slice reuse")
	}
}

func TestOutcomesDefinedAndMasteryScale(t *testing.T) {
	f, _ := findRule(t, ItemOutcomesDefined).Evaluate(context.Background(), CourseSnapshot{
		Outcomes: []OutcomeSnap{{Title: "A"}, {Title: "B"}},
	})
	if f.Status != StatusInProgress || f.Progress == nil || f.Progress.Done != 2 {
		t.Fatalf("defined=%+v", f)
	}

	scale := json.RawMessage(`{"levels":[{"level":4,"label":"Exceeds","minScore":3.5},{"level":3,"label":"Meets","minScore":2.5}],"masteryThreshold":3}`)
	f, _ = findRule(t, ItemOutcomesMasteryScale).Evaluate(context.Background(), CourseSnapshot{
		SbgEnabled: true, SbgProficiencyScaleJSON: scale,
	})
	if f.Status != StatusDone {
		t.Fatalf("mastery=%s", f.Status)
	}
}

func TestOutcomesEvidenceHasNoUserPII(t *testing.T) {
	mod := uuid.New()
	a := uuid.New()
	pts := 10
	snap := CourseSnapshot{
		StructureItems: []StructureItem{
			{ID: mod, Kind: "module", Title: "M"},
			{ID: a, Kind: "assignment", Title: "Essay", ParentID: &mod},
		},
		ItemMeta:         map[uuid.UUID]ItemMeta{a: {PointsWorth: &pts}},
		GradableItems:    []GradableItem{{ID: a, Title: "Essay", Kind: "assignment", ParentID: &mod, Points: &pts}},
		GradableComputed: true,
		Outcomes:         []OutcomeSnap{{ID: uuid.New(), Title: "Write clearly"}},
		People: []PersonSnap{
			{DisplayName: "Secret Student", Role: "student"},
		},
	}
	for _, id := range []ItemID{
		ItemOutcomesAssessmentMapping,
		ItemOutcomesCoverage,
		ItemStructureEmptyModules,
	} {
		f, err := findRule(t, id).Evaluate(context.Background(), snap)
		if err != nil {
			t.Fatal(err)
		}
		blob := f.DetailDefault
		for _, e := range f.Evidence {
			blob += e.Label + e.Sublabel
		}
		if strings.Contains(blob, "Secret Student") {
			t.Fatalf("%s leaked person name", id)
		}
	}
}
