package gradingagent

import "testing"

func TestEstimateRunCost_noSample(t *testing.T) {
	got := EstimateRunCost(24, nil)
	if got.SubmissionCount != 24 || got.HasSample {
		t.Fatalf("got=%+v", got)
	}
}

func TestEstimateRunCost_withCostSample(t *testing.T) {
	perSubmission := 0.002
	sample := &DryRunCostSample{CostUSD: &perSubmission, PromptTokens: intPtr(100), CompletionTokens: intPtr(50)}
	got := EstimateRunCost(10, sample)
	if !got.HasSample || got.TokensOnly {
		t.Fatalf("expected cost estimate, got=%+v", got)
	}
	if got.PromptTokens == nil || *got.PromptTokens != 1000 {
		t.Fatalf("prompt tokens=%v", got.PromptTokens)
	}
	if got.CostMinUSD == nil || got.CostMaxUSD == nil {
		t.Fatalf("missing cost range: %+v", got)
	}
	base := perSubmission * 10
	wantMin := roundCostUSD(base * costEstimateRangeLow)
	wantMax := roundCostUSD(base * costEstimateRangeHigh)
	if *got.CostMinUSD != wantMin || *got.CostMaxUSD != wantMax {
		t.Fatalf("range=%v-%v want=%v-%v", *got.CostMinUSD, *got.CostMaxUSD, wantMin, wantMax)
	}
}

func TestEstimateRunCost_tokensOnlyWhenPriceMissing(t *testing.T) {
	sample := &DryRunCostSample{PromptTokens: intPtr(200), CompletionTokens: intPtr(80)}
	got := EstimateRunCost(5, sample)
	if !got.HasSample || !got.TokensOnly {
		t.Fatalf("got=%+v", got)
	}
	if got.CostMinUSD != nil || got.CostMaxUSD != nil {
		t.Fatalf("unexpected cost: %+v", got)
	}
}

func intPtr(v int) *int { return &v }
