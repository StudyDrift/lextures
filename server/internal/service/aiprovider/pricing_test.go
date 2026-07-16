package aiprovider

import "testing"

func TestEstimateCostUSD_OpenAI(t *testing.T) {
	t.Parallel()
	cost, ok := EstimateCostUSD(ProviderOpenAI, "gpt-4o-mini", 1_000_000, 1_000_000)
	if !ok {
		t.Fatal("expected rates")
	}
	if cost <= 0 {
		t.Fatalf("cost=%v", cost)
	}
	// 0.15 + 0.60 = 0.75
	if cost < 0.74 || cost > 0.76 {
		t.Fatalf("cost=%v want ~0.75", cost)
	}
}

func TestApplyCostEstimate_SkipsWhenProviderCostPresent(t *testing.T) {
	t.Parallel()
	u := UsageInfo{PromptTokens: 100, CompletionTokens: 50, CostUSD: 0.02}
	if ApplyCostEstimate(ProviderOpenAI, "gpt-4o-mini", &u) {
		t.Fatal("should not estimate when cost present")
	}
	if u.CostUSD != 0.02 {
		t.Fatalf("cost changed: %v", u.CostUSD)
	}
}

func TestApplyCostEstimate_FillsMissingCost(t *testing.T) {
	t.Parallel()
	u := UsageInfo{PromptTokens: 1_000_000, CompletionTokens: 0}
	if !ApplyCostEstimate(ProviderOpenAI, "gpt-4o-mini", &u) {
		t.Fatal("expected estimate")
	}
	if u.CostUSD <= 0 {
		t.Fatalf("cost=%v", u.CostUSD)
	}
}

func TestEstimateCostUSD_UnknownModel(t *testing.T) {
	t.Parallel()
	_, ok := EstimateCostUSD(ProviderOpenAI, "totally-unknown-model", 100, 100)
	if ok {
		t.Fatal("expected no rates")
	}
}
