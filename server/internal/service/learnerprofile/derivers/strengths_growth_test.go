package derivers

import (
	"math"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/lextures/lextures/server/internal/repos/learnermodel"
)

func TestClassifyConcept_Strength(t *testing.T) {
	agg := aggregatedConcept{
		Name:             "Linear equations",
		StoredMastery:    0.92,
		EffectiveMastery: 0.92,
		AttemptCount:     5,
	}
	cat, ok := classifyConcept(agg, time.Now().UTC())
	if !ok || cat != categoryStrength {
		t.Fatalf("cat=%v ok=%v want strength", cat, ok)
	}
}

func TestClassifyConcept_DecayedToNeedsReview(t *testing.T) {
	now := time.Date(2026, 4, 15, 12, 0, 0, 0, time.UTC)
	lastSeen := now.AddDate(0, 0, -40)
	stored := 0.9
	effective := learnermodel.DecayAdjustedMasteryAt(stored, &lastSeen, 0.02, now)
	agg := aggregatedConcept{
		Name:             "Factoring",
		StoredMastery:    stored,
		EffectiveMastery: effective,
		AttemptCount:     4,
		LastSeenAt:       &lastSeen,
		LastSeenDays:     40,
	}
	cat, ok := classifyConcept(agg, now)
	if !ok || cat != categoryNeedsReview {
		t.Fatalf("cat=%v ok=%v want needs_review (effective=%v)", cat, ok, effective)
	}
}

func TestClassifyConcept_LowMasteryGrowth(t *testing.T) {
	agg := aggregatedConcept{
		Name:             "Unit conversions",
		StoredMastery:    0.41,
		EffectiveMastery: 0.41,
		AttemptCount:     3,
	}
	cat, ok := classifyConcept(agg, time.Now().UTC())
	if !ok || cat != categoryGrowth {
		t.Fatalf("cat=%v ok=%v want growth", cat, ok)
	}
}

func TestComputeStrengthsGrowth_InsufficientData(t *testing.T) {
	now := time.Date(2026, 4, 15, 12, 0, 0, 0, time.UTC)
	rows := []conceptCourseRow{{
		Slug: "only-one", Name: "Only", ConceptID: uuid.New(), CourseID: uuid.New(),
		DecayLambda: 0.02, StoredMastery: 0.7, AttemptCount: 2,
	}}
	_, sufficient, count := computeStrengthsGrowth(strengthsGrowthComputeInput{
		ConceptRows: rows,
		Now:         now,
	})
	if sufficient {
		t.Fatal("expected insufficient data")
	}
	if count != 1 {
		t.Fatalf("count=%d want 1", count)
	}
}

func TestComputeStrengthsGrowth_StrengthAcrossTwoCourses(t *testing.T) {
	now := time.Date(2026, 4, 15, 12, 0, 0, 0, time.UTC)
	lastSeen := now
	courseA, courseB := uuid.New(), uuid.New()
	rows := []conceptCourseRow{
		{Slug: "linear-equations", Name: "Linear equations", ConceptID: uuid.New(), CourseID: courseA,
			DecayLambda: 0.02, StoredMastery: 0.92, AttemptCount: 3, LastSeenAt: &lastSeen},
		{Slug: "linear-equations", Name: "Linear equations", ConceptID: uuid.New(), CourseID: courseB,
			DecayLambda: 0.02, StoredMastery: 0.92, AttemptCount: 4, LastSeenAt: &lastSeen},
		{Slug: "fractions", Name: "Fractions", ConceptID: uuid.New(), CourseID: courseA,
			DecayLambda: 0.02, StoredMastery: 0.35, AttemptCount: 2, LastSeenAt: &lastSeen},
		{Slug: "decimals", Name: "Decimals", ConceptID: uuid.New(), CourseID: courseA,
			DecayLambda: 0.02, StoredMastery: 0.4, AttemptCount: 2, LastSeenAt: &lastSeen},
	}
	summary, sufficient, _ := computeStrengthsGrowth(strengthsGrowthComputeInput{
		ConceptRows: rows,
		Now:         now,
	})
	if !sufficient {
		t.Fatal("expected sufficient data")
	}
	if len(summary.Strengths) == 0 || summary.Strengths[0].Concept != "Linear equations" {
		t.Fatalf("strengths=%+v", summary.Strengths)
	}
	if summary.Strengths[0].Courses != 2 {
		t.Fatalf("courses=%d want 2", summary.Strengths[0].Courses)
	}
	wantMastery := learnermodel.DecayAdjustedMasteryAt(0.92, &lastSeen, 0.02, now)
	if math.Abs(summary.Strengths[0].Mastery-wantMastery) > 0.01 {
		t.Fatalf("mastery=%v want ~%v", summary.Strengths[0].Mastery, wantMastery)
	}
}

func TestComputeStrengthsGrowth_CapsAtFive(t *testing.T) {
	now := time.Date(2026, 4, 15, 12, 0, 0, 0, time.UTC)
	lastSeen := now.AddDate(0, 0, -1)
	rows := make([]conceptCourseRow, 0, 200)
	for i := 0; i < 200; i++ {
		rows = append(rows, conceptCourseRow{
			Slug:          uuid.New().String(),
			Name:          "Weak concept " + uuid.New().String(),
			ConceptID:     uuid.New(),
			CourseID:      uuid.New(),
			DecayLambda:   0.02,
			StoredMastery: 0.2,
			AttemptCount:  2,
			LastSeenAt:    &lastSeen,
		})
	}
	summary, sufficient, _ := computeStrengthsGrowth(strengthsGrowthComputeInput{
		ConceptRows: rows,
		Now:         now,
	})
	if !sufficient {
		t.Fatal("expected sufficient data")
	}
	if len(summary.Growth) > strengthsGrowthMaxPerCategory {
		t.Fatalf("growth=%d want <= %d", len(summary.Growth), strengthsGrowthMaxPerCategory)
	}
}

func TestComputeStrengthsGrowth_MisconceptionGrowthItem(t *testing.T) {
	now := time.Date(2026, 4, 15, 12, 0, 0, 0, time.UTC)
	lastSeen := now.AddDate(0, 0, -1)
	desc := "Treats % as additive"
	concept := "Percentages"
	rows := make([]conceptCourseRow, 0, 3)
	for i := 0; i < 3; i++ {
		rows = append(rows, conceptCourseRow{
			Slug: "concept-" + string(rune('a'+i)), Name: "Concept " + string(rune('a'+i)), ConceptID: uuid.New(), CourseID: uuid.New(),
			DecayLambda: 0.02, StoredMastery: 0.6, AttemptCount: 2, LastSeenAt: &lastSeen,
		})
	}
	summary, sufficient, _ := computeStrengthsGrowth(strengthsGrowthComputeInput{
		ConceptRows: rows,
		Misconceptions: []misconceptionRow{{
			MisconceptionID: uuid.New(),
			Name:            "Treats % as additive",
			Description:     &desc,
			ConceptName:     &concept,
			CourseID:        uuid.New(),
			TriggerCount:    3,
		}},
		Now: now,
	})
	if !sufficient {
		t.Fatal("expected sufficient data")
	}
	found := false
	for _, item := range summary.Growth {
		if item.Misconception == "Treats % as additive" {
			found = true
			if item.Description != desc {
				t.Fatalf("description=%q", item.Description)
			}
		}
	}
	if !found {
		t.Fatalf("growth=%+v", summary.Growth)
	}
}