package adaptivecontent

import (
	"testing"

	"github.com/google/uuid"

	acrepo "github.com/lextures/lextures/server/internal/repos/adaptivecontent"
)

func TestComputeLift(t *testing.T) {
	pre := float32(40)
	post := float32(75)
	lift := ComputeLift(&pre, &post)
	if lift == nil || *lift != 35 {
		t.Fatalf("expected lift 35, got %v", lift)
	}
	if ComputeLift(nil, &post) != nil {
		t.Fatal("nil pre should yield nil lift")
	}
	if ComputeLift(&pre, nil) != nil {
		t.Fatal("nil post should yield nil lift")
	}
}

func TestDiffInMeans_InsufficientData(t *testing.T) {
	treat := make([]float64, MinNPerArm-1)
	hold := make([]float64, MinNPerArm)
	for i := range treat {
		treat[i] = 20
	}
	for i := range hold {
		hold[i] = 5
	}
	r := DiffInMeans(treat, hold, nil, nil)
	if r.Verdict != VerdictInsufficientData {
		t.Fatalf("expected insufficient_data, got %s", r.Verdict)
	}
}

func TestDiffInMeans_Helping(t *testing.T) {
	treat := make([]float64, MinNPerArm)
	hold := make([]float64, MinNPerArm)
	for i := range treat {
		treat[i] = 25
		hold[i] = 10
	}
	r := DiffInMeans(treat, hold, nil, nil)
	if r.Verdict != VerdictHelping {
		t.Fatalf("expected helping, got %s (diff=%v)", r.Verdict, r.Diff)
	}
	if r.Diff < HelpingMarginPts {
		t.Fatalf("diff too small: %v", r.Diff)
	}
}

func TestDiffInMeans_Regressing(t *testing.T) {
	treat := make([]float64, MinNPerArm)
	hold := make([]float64, MinNPerArm)
	for i := range treat {
		treat[i] = 5
		hold[i] = 20
	}
	r := DiffInMeans(treat, hold, nil, nil)
	if r.Verdict != VerdictRegressing {
		t.Fatalf("expected regressing, got %s (diff=%v)", r.Verdict, r.Diff)
	}
}

func TestDiffInMeans_NoEffect(t *testing.T) {
	treat := make([]float64, MinNPerArm)
	hold := make([]float64, MinNPerArm)
	for i := range treat {
		treat[i] = 12
		hold[i] = 10
	}
	r := DiffInMeans(treat, hold, nil, nil)
	if r.Verdict != VerdictNoEffect {
		t.Fatalf("expected no_effect, got %s (diff=%v)", r.Verdict, r.Diff)
	}
}

func TestDiffInMeans_StdError(t *testing.T) {
	treat := []float64{10, 20, 30, 40, 50, 60, 70, 80, 90, 100}
	hold := []float64{0, 5, 10, 15, 20, 25, 30, 35, 40, 45}
	r := DiffInMeans(treat, hold, nil, nil)
	if r.StdError <= 0 {
		t.Fatalf("expected positive SE, got %v", r.StdError)
	}
	if r.Treatment.N != 10 || r.Holdout.N != 10 {
		t.Fatalf("n mismatch: %+v", r)
	}
}

func TestSuppressSmallCell(t *testing.T) {
	m := float32(12.5)
	if SuppressSmallCell(SmallCellMinN-1, &m) != nil {
		t.Fatal("expected suppression below k")
	}
	got := SuppressSmallCell(SmallCellMinN, &m)
	if got == nil || *got != m {
		t.Fatal("expected mean retained at k")
	}
}

func TestAggregateByMode_IdentifiesWeaker(t *testing.T) {
	vid := uuid.New()
	samples := []acrepo.OutcomeLiftSample{
		{Lift: 30, WasHoldout: false, EmphasisMode: "remediate", VariantID: &vid},
		{Lift: 28, WasHoldout: false, EmphasisMode: "remediate", VariantID: &vid},
		{Lift: 26, WasHoldout: false, EmphasisMode: "remediate", VariantID: &vid},
		{Lift: 24, WasHoldout: false, EmphasisMode: "remediate", VariantID: &vid},
		{Lift: 22, WasHoldout: false, EmphasisMode: "remediate", VariantID: &vid},
		{Lift: 5, WasHoldout: false, EmphasisMode: "compress", VariantID: &vid},
		{Lift: 4, WasHoldout: false, EmphasisMode: "compress", VariantID: &vid},
		{Lift: 3, WasHoldout: false, EmphasisMode: "compress", VariantID: &vid},
		{Lift: 2, WasHoldout: false, EmphasisMode: "compress", VariantID: &vid},
		{Lift: 1, WasHoldout: false, EmphasisMode: "compress", VariantID: &vid},
		{Lift: 10, WasHoldout: true, EmphasisMode: "compress"}, // ignored for mode
	}
	modes := AggregateByMode(samples)
	if len(modes) != 2 {
		t.Fatalf("expected 2 modes, got %d", len(modes))
	}
	by := map[string]ModeAggregate{}
	for _, m := range modes {
		by[m.EmphasisMode] = m
	}
	rem := by["remediate"]
	comp := by["compress"]
	if rem.MeanLift == nil || comp.MeanLift == nil {
		t.Fatal("means should not be suppressed at n=5")
	}
	if *rem.MeanLift <= *comp.MeanLift {
		t.Fatalf("remediate should be stronger: rem=%v comp=%v", *rem.MeanLift, *comp.MeanLift)
	}
}

func TestAggregateByVariant_SuppressesTiny(t *testing.T) {
	vid := uuid.New()
	samples := []acrepo.OutcomeLiftSample{
		{Lift: 10, WasHoldout: false, EmphasisMode: "remediate", VariantID: &vid},
		{Lift: 12, WasHoldout: false, EmphasisMode: "remediate", VariantID: &vid},
	}
	vars := AggregateByVariant(samples)
	if len(vars) != 1 {
		t.Fatalf("expected 1 variant, got %d", len(vars))
	}
	if vars[0].MeanLift != nil {
		t.Fatal("n=2 should suppress mean")
	}
	if vars[0].N != 2 {
		t.Fatalf("n=%d", vars[0].N)
	}
}

func TestSampleVariance_Single(t *testing.T) {
	if SampleVariance([]float64{5}) != 0 {
		t.Fatal("single sample variance must be 0")
	}
}
