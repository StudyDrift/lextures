package analytics

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestProjectNoopProbe(t *testing.T) {
	raw := 1.0
	max := 1.0
	state, _ := json.Marshal(map[string]any{"response": "paris", "attempts": 2})
	s := Project(ProjectInput{
		ToolID:    "noop_probe",
		StateJSON: state,
		Status:    "completed",
		ScoreRaw:  &raw,
		ScoreMax:  &max,
	})
	if !s.Engaged || !s.Completed {
		t.Fatalf("expected engaged+completed: %+v", s)
	}
	if s.ScorePct == nil || *s.ScorePct != 100 {
		t.Fatalf("scorePct: %+v", s.ScorePct)
	}
	if s.Facets["attempts"] != 2 {
		t.Fatalf("attempts facet: %#v", s.Facets["attempts"])
	}
	if s.Facets["correct"] != true {
		t.Fatalf("correct facet: %#v", s.Facets["correct"])
	}
}

func TestProjectDefaultDuration(t *testing.T) {
	start := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	end := start.Add(5 * time.Second)
	s := Project(ProjectInput{
		ToolID:            "sandbox_probe",
		StateJSON:         json.RawMessage(`{}`),
		Status:            "in_progress",
		FirstInteractedAt: &start,
		LastInteractedAt:  &end,
	})
	if !s.Engaged || s.Completed {
		t.Fatalf("unexpected: %+v", s)
	}
	if s.DurationMs == nil || *s.DurationMs != 5000 {
		t.Fatalf("duration: %+v", s.DurationMs)
	}
}

func TestAggregateInstance_RoleFilterAndSuppression(t *testing.T) {
	rows := []SummaryRow{
		{EnrollmentID: "e1", DisplayName: "A", Role: "student", Engaged: true, Completed: true, ScorePct: f64(80)},
		{EnrollmentID: "e2", DisplayName: "B", Role: "student", Engaged: true, Completed: false},
		{EnrollmentID: "e3", DisplayName: "C", Role: "student", Engaged: false, Status: "not_started"},
		{EnrollmentID: "t1", DisplayName: "Teach", Role: "teacher", Engaged: true, Completed: true, ScorePct: f64(100)},
	}
	agg := AggregateInstance(rows, "noop_probe", 5, true)
	if agg.Learners != 3 || agg.Engaged != 2 || agg.Completed != 1 {
		t.Fatalf("counts: %+v", agg)
	}
	if !agg.Suppressed {
		t.Fatal("expected suppressed for n<5")
	}
	if len(agg.NeedsAttention) == 0 {
		t.Fatal("needsAttention should still list learners")
	}
	full := AggregateInstance(rows, "noop_probe", 5, false)
	if full.Suppressed || full.ScoreMean == nil {
		t.Fatalf("roster surface should not suppress: %+v", full)
	}
}

func TestAggregateInstance_FacetDistribution(t *testing.T) {
	rows := []SummaryRow{
		{EnrollmentID: "e1", Role: "student", Engaged: true, Completed: true, Facets: map[string]any{"correct": true, "attempts": 1}},
		{EnrollmentID: "e2", Role: "student", Engaged: true, Completed: true, Facets: map[string]any{"correct": false, "attempts": 2}},
		{EnrollmentID: "e3", Role: "student", Engaged: true, Completed: true, Facets: map[string]any{"correct": false, "attempts": 2}},
		{EnrollmentID: "e4", Role: "student", Engaged: true, Completed: true, Facets: map[string]any{"correct": false, "attempts": 3}},
		{EnrollmentID: "e5", Role: "student", Engaged: true, Completed: true, Facets: map[string]any{"correct": true, "attempts": 1}},
	}
	agg := AggregateInstance(rows, "noop_probe", 5, false)
	var correct FacetDistribution
	for _, f := range agg.Facets {
		if f.Key == "correct" {
			correct = f
		}
	}
	if len(correct.Values) != 2 {
		t.Fatalf("correct values: %+v", correct.Values)
	}
	if correct.Values[0].Value != "false" || correct.Values[0].Count != 3 {
		t.Fatalf("expected false=3 first: %+v", correct.Values)
	}
}

func TestCacheInvalidation(t *testing.T) {
	c := NewAggregateCache(time.Hour)
	c.Set(CacheKeyInstance("i1"), "v")
	c.InvalidateExact(CacheKeyInstance("i1"))
	if _, ok := c.Get(CacheKeyInstance("i1")); ok {
		t.Fatal("expected miss after invalidate")
	}
	c.Set("inst:abc", 1)
	c.InvalidatePrefix("inst:")
	if _, ok := c.Get("inst:abc"); ok {
		t.Fatal("prefix invalidate failed")
	}
}

func TestPointsFromScorePct(t *testing.T) {
	s := 50.0
	if got := PointsFromScorePct(&s, 10); got != 5 {
		t.Fatalf("got %v", got)
	}
	if PointsFromScorePct(nil, 10) != 0 {
		t.Fatal("nil score")
	}
}

func TestClassifyBridgedGradeEffect(t *testing.T) {
	id := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	if ClassifyBridgedGradeEffect(id, false, true) != GradeActionUnchanged {
		t.Fatal("unlinked")
	}
	if ClassifyBridgedGradeEffect(id, true, true) != GradeActionReverted {
		t.Fatal("linked+score")
	}
}

func TestToolDisengagePct(t *testing.T) {
	rows := []SummaryRow{
		{Role: "student", Engaged: true},
		{Role: "student", Engaged: false},
		{Role: "teacher", Engaged: false},
	}
	pct := ToolDisengagePct(rows)
	if pct != 50 {
		t.Fatalf("got %v", pct)
	}
	var got float32
	ApplyToolDisengageSignal(rows, func(p float32) { got = p })
	if got != 50 {
		t.Fatalf("apply got %v", got)
	}
}

func f64(v float64) *float64 { return &v }
