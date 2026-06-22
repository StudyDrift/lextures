package gradecurve

import (
	"testing"

	"github.com/google/uuid"
)

func scores(vals ...float64) []ScoreInput {
	out := make([]ScoreInput, len(vals))
	for i, v := range vals {
		out[i] = ScoreInput{StudentID: uuid.New(), RawScore: v}
	}
	return out
}

func TestFlatBonusCapsAtMax(t *testing.T) {
	in := scores(90, 95)
	prev, err := Preview(in, Options{
		MaxPoints:     100,
		AllowAboveMax: false,
		Method:        MethodFlatBonus,
		Params:        Params{Bonus: ptrFloat(10)},
	})
	if err != nil {
		t.Fatal(err)
	}
	if prev.MeanAfter == nil || *prev.MeanAfter != 100 {
		t.Fatalf("expected mean 100 after cap, got %v", prev.MeanAfter)
	}
	for _, r := range prev.Results {
		if r.AdjustedScore > 100 {
			t.Fatalf("score %v exceeded max", r.AdjustedScore)
		}
	}
}

func TestLinearScaleTargetMean(t *testing.T) {
	in := scores(50, 60, 70, 80)
	prev, err := Preview(in, Options{
		MaxPoints:     100,
		AllowAboveMax: false,
		Method:        MethodLinearScale,
		Params:        Params{TargetMean: ptrFloat(75)},
	})
	if err != nil {
		t.Fatal(err)
	}
	if prev.MeanBefore == nil || *prev.MeanBefore != 65 {
		t.Fatalf("mean before: got %v want 65", prev.MeanBefore)
	}
	if prev.MeanAfter == nil || *prev.MeanAfter != 75 {
		t.Fatalf("mean after: got %v want 75", prev.MeanAfter)
	}
}

func TestSetMinimum(t *testing.T) {
	in := scores(30, 50, 70)
	prev, err := Preview(in, Options{
		MaxPoints: 100,
		Method:    MethodSetMinimum,
		Params:    Params{Minimum: ptrFloat(50)},
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range prev.Results {
		if r.RawScore < 50 && r.AdjustedScore != 50 {
			t.Fatalf("expected floor 50 for raw %v, got %v", r.RawScore, r.AdjustedScore)
		}
		if r.RawScore >= 50 && r.AdjustedScore != r.RawScore {
			t.Fatalf("expected unchanged score for raw %v, got %v", r.RawScore, r.AdjustedScore)
		}
	}
}

func TestExcusedExcluded(t *testing.T) {
	s1 := uuid.New()
	s2 := uuid.New()
	in := []ScoreInput{
		{StudentID: s1, RawScore: 40},
		{StudentID: s2, RawScore: 80, Excused: true},
	}
	prev, err := Preview(in, Options{
		MaxPoints: 100,
		Method:    MethodLinearScale,
		Params:    Params{TargetMean: ptrFloat(75)},
	})
	if err != nil {
		t.Fatal(err)
	}
	if prev.EligibleCount != 1 {
		t.Fatalf("eligible count: got %d want 1", prev.EligibleCount)
	}
	if len(prev.Results) != 1 {
		t.Fatalf("results len: got %d want 1", len(prev.Results))
	}
}

func TestSqrtCurve(t *testing.T) {
	in := scores(25, 100)
	prev, err := Preview(in, Options{
		MaxPoints: 100,
		Method:    MethodSqrtCurve,
	})
	if err != nil {
		t.Fatal(err)
	}
	byRaw := make(map[float64]float64, len(prev.Results))
	for _, r := range prev.Results {
		byRaw[r.RawScore] = r.AdjustedScore
	}
	if byRaw[25] != 50 {
		t.Fatalf("sqrt(25/100)*100 = 50, got %v", byRaw[25])
	}
	if byRaw[100] != 100 {
		t.Fatalf("sqrt(100/100)*100 = 100, got %v", byRaw[100])
	}
}

func TestAllowAboveMax(t *testing.T) {
	in := scores(95)
	prev, err := Preview(in, Options{
		MaxPoints:     100,
		AllowAboveMax: true,
		Method:        MethodFlatBonus,
		Params:        Params{Bonus: ptrFloat(10)},
	})
	if err != nil {
		t.Fatal(err)
	}
	if prev.Results[0].AdjustedScore != 105 {
		t.Fatalf("expected 105, got %v", prev.Results[0].AdjustedScore)
	}
}

func ptrFloat(v float64) *float64 { return &v }
