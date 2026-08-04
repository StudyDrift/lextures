package coursechecklist

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus/testutil"
)

func testRegistry(t *testing.T, items ...ItemDescriptor) *Registry {
	t.Helper()
	reg, err := NewRegistry(items)
	if err != nil {
		t.Fatalf("NewRegistry: %v", err)
	}
	return reg
}

func TestEvaluateTwoRulesOrderedCounts(t *testing.T) {
	reg := testRegistry(t, referenceCourseDates(), referenceCourseSections())
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	snap := CourseSnapshot{
		StartsAt:        &start,
		EndsAt:          &end,
		SectionsEnabled: false,
	}
	res := Evaluate(context.Background(), snap, EvaluateOptions{Registry: reg})
	if len(res.Findings) != 2 {
		t.Fatalf("findings=%d want 2", len(res.Findings))
	}
	if res.Findings[0].ID != ItemCourseDates || res.Findings[1].ID != ItemCourseSections {
		t.Fatalf("order: %v, %v", res.Findings[0].ID, res.Findings[1].ID)
	}
	if res.Findings[0].Finding.Status != StatusDone {
		t.Fatalf("dates status=%s", res.Findings[0].Finding.Status)
	}
	if res.Findings[1].Finding.Status != StatusNotApplicable {
		t.Fatalf("sections status=%s", res.Findings[1].Finding.Status)
	}
	if res.Counts.Done != 1 || res.Counts.NotApplicable != 1 || res.Counts.Total != 1 {
		t.Fatalf("counts=%+v", res.Counts)
	}
	if res.Counts.OutstandingEssential != 0 {
		t.Fatalf("outstanding=%d", res.Counts.OutstandingEssential)
	}
}

func TestEvaluatePanicContained(t *testing.T) {
	panicRule := referenceCourseDates()
	panicRule.ID = "ref.panic"
	panicRule.TitleKey = "coursechecklist.item.ref.panic.title"
	panicRule.WhyKey = "coursechecklist.item.ref.panic.why"
	panicRule.Evaluate = func(context.Context, CourseSnapshot) (Finding, error) {
		panic("boom")
	}
	okRule := referenceCourseSections()
	reg := testRegistry(t, panicRule, okRule)

	before := testutil.ToFloat64(ruleErrorsCounter().WithLabelValues("ref.panic", "panic"))
	res := Evaluate(context.Background(), CourseSnapshot{SectionsEnabled: true, Sections: []SectionSnap{{SectionCode: "A"}}}, EvaluateOptions{Registry: reg})
	after := testutil.ToFloat64(ruleErrorsCounter().WithLabelValues("ref.panic", "panic"))
	if after-before != 1 {
		t.Fatalf("panic counter delta=%v", after-before)
	}
	if res.Findings[0].Finding.Status != StatusUnknown {
		t.Fatalf("panic status=%s", res.Findings[0].Finding.Status)
	}
	if res.Findings[1].Finding.Status != StatusDone {
		t.Fatalf("other status=%s", res.Findings[1].Finding.Status)
	}
}

func TestEvaluateSectionsNotApplicable(t *testing.T) {
	reg := testRegistry(t, referenceCourseSections())
	res := Evaluate(context.Background(), CourseSnapshot{SectionsEnabled: false}, EvaluateOptions{Registry: reg})
	if res.Findings[0].Finding.Status != StatusNotApplicable {
		t.Fatalf("status=%s", res.Findings[0].Finding.Status)
	}
	if res.Counts.Total != 0 || res.Counts.NotApplicable != 1 {
		t.Fatalf("counts=%+v", res.Counts)
	}
}

func TestEvidenceTruncation(t *testing.T) {
	rule := referenceCourseSections()
	rule.Evaluate = func(context.Context, CourseSnapshot) (Finding, error) {
		ev := make([]EvidenceRow, 500)
		for i := range ev {
			ev[i] = EvidenceRow{Label: "x"}
		}
		return Finding{Status: StatusTodo, Evidence: ev}, nil
	}
	reg := testRegistry(t, rule)
	res := Evaluate(context.Background(), CourseSnapshot{SectionsEnabled: true}, EvaluateOptions{Registry: reg})
	f := res.Findings[0].Finding
	if len(f.Evidence) != 200 || f.TruncatedAt == nil || *f.TruncatedAt != 200 {
		t.Fatalf("evidence=%d truncated=%v", len(f.Evidence), f.TruncatedAt)
	}
}

func TestEvaluateOnlySingleItem(t *testing.T) {
	var evaluated []ItemID
	dates := referenceCourseDates()
	dates.Evaluate = func(ctx context.Context, snap CourseSnapshot) (Finding, error) {
		evaluated = append(evaluated, ItemCourseDates)
		return evalCourseDates(ctx, snap)
	}
	sections := referenceCourseSections()
	sections.Evaluate = func(ctx context.Context, snap CourseSnapshot) (Finding, error) {
		evaluated = append(evaluated, ItemCourseSections)
		return evalCourseSections(ctx, snap)
	}
	reg := testRegistry(t, dates, sections)

	res := Evaluate(context.Background(), CourseSnapshot{}, EvaluateOptions{
		Registry: reg,
		Only:     []ItemID{ItemCourseDates},
	})
	if len(res.Findings) != 1 || res.Findings[0].ID != ItemCourseDates {
		t.Fatalf("findings=%v", res.Findings)
	}
	if len(evaluated) != 1 || evaluated[0] != ItemCourseDates {
		t.Fatalf("evaluated=%v", evaluated)
	}
	needs := DataNeedsForEvaluate(reg, EvaluateOptions{Only: []ItemID{ItemCourseDates}})
	if len(needs) != 1 || needs[0] != DataNeedCourse {
		t.Fatalf("needs=%v", needs)
	}
}

func TestEvaluateDeterminism(t *testing.T) {
	reg := testRegistry(t, referenceCourseDates(), referenceCourseSections())
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	snap := CourseSnapshot{StartsAt: &start, SectionsEnabled: false}
	a := Evaluate(context.Background(), snap, EvaluateOptions{Registry: reg})
	b := Evaluate(context.Background(), snap, EvaluateOptions{Registry: reg})
	if a.CatalogVersion != b.CatalogVersion || a.CatalogVersion == "" {
		t.Fatalf("catalog versions %q vs %q", a.CatalogVersion, b.CatalogVersion)
	}
	ja, err := json.Marshal(a)
	if err != nil {
		t.Fatal(err)
	}
	jb, err := json.Marshal(b)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(ja, jb) {
		t.Fatalf("results not byte-identical\n%s\n%s", ja, jb)
	}
}

func TestLazyLoaderTimeoutUnknown(t *testing.T) {
	const lazyID LazyLoaderID = "slow.probe"
	rule := referenceCourseDates()
	rule.ID = "ref.lazy"
	rule.TitleKey = "coursechecklist.item.ref.lazy.title"
	rule.WhyKey = "coursechecklist.item.ref.lazy.why"
	rule.LazyNeeds = []LazyLoaderID{lazyID}
	rule.Evaluate = func(context.Context, CourseSnapshot) (Finding, error) {
		return Finding{Status: StatusDone, DetailDefault: "ok"}, nil
	}
	other := referenceCourseSections()
	reg := testRegistry(t, rule, other)

	loader := LazyFunc{
		LoaderID: lazyID,
		Fn: func(ctx context.Context, snap *CourseSnapshot) error {
			<-ctx.Done()
			return ctx.Err()
		},
	}
	start := time.Now()
	res := Evaluate(context.Background(), CourseSnapshot{SectionsEnabled: false}, EvaluateOptions{
		Registry:    reg,
		LazyLoaders: map[LazyLoaderID]LazyLoader{lazyID: loader},
	})
	elapsed := time.Since(start)
	if elapsed > 6*time.Second {
		t.Fatalf("evaluation blocked too long: %v", elapsed)
	}
	if res.Findings[0].Finding.Status != StatusUnknown {
		t.Fatalf("lazy status=%s", res.Findings[0].Finding.Status)
	}
	if res.Findings[1].Finding.Status != StatusNotApplicable {
		t.Fatalf("other status=%s", res.Findings[1].Finding.Status)
	}
}

func TestEvidenceHasNoEmailFields(t *testing.T) {
	reg := testRegistry(t, referenceCourseSections())
	res := Evaluate(context.Background(), CourseSnapshot{
		SectionsEnabled: true,
		Sections:        []SectionSnap{{SectionCode: "A", Name: "Alpha"}},
	}, EvaluateOptions{Registry: reg})
	raw, err := json.Marshal(res.Findings[0].Finding.Evidence)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(strings.ToLower(string(raw)), "email") || strings.Contains(string(raw), "@") {
		t.Fatalf("evidence leaked email-like data: %s", raw)
	}
	// PersonSnap JSON tags must not include email/DOB.
	type probe struct {
		PersonSnap
	}
	b, _ := json.Marshal(probe{})
	if strings.Contains(strings.ToLower(string(b)), "email") || strings.Contains(strings.ToLower(string(b)), "dob") {
		t.Fatalf("PersonSnap has forbidden fields: %s", b)
	}
}

func TestCatalogVersionStable(t *testing.T) {
	reg := testRegistry(t, referenceCourseDates(), referenceCourseSections())
	a := catalogVersionFor(reg)
	b := catalogVersionFor(reg)
	if a != b || len(a) != 16 {
		t.Fatalf("catalog version %q / %q", a, b)
	}
	if EngineVersion() != 1 {
		t.Fatalf("engine version=%d", EngineVersion())
	}
}
