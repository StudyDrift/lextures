package aiusage

import (
	"testing"

	"github.com/lextures/lextures/server/internal/service/aiprovider"
)

func TestInsert_defaultsEmptyProviderToUnknown(t *testing.T) {
	t.Parallel()
	e := Entry{Feature: "ai_tutor", Model: "gpt-4o-mini", Provider: ""}
	// Mirror Insert normalization without a pool.
	provider := e.Provider
	if provider == "" {
		provider = defaultProviderUnknown
	}
	if provider != "unknown" {
		t.Fatalf("provider=%q", provider)
	}
}

func TestEntryFromCallMeta_estimatesMissingCost(t *testing.T) {
	t.Parallel()
	meta := aiprovider.CallMeta{
		Provider:   aiprovider.ProviderOpenAI,
		ModelID:    "gpt-4o-mini",
		ModelAlias: "text-fast",
	}
	usage := aiprovider.UsageInfo{PromptTokens: 1_000_000, CompletionTokens: 0}
	e := EntryFromCallMeta(nil, nil, "translation", meta, usage, true)
	if e.Provider != "openai" {
		t.Fatalf("provider=%q", e.Provider)
	}
	if !e.CostEstimated {
		t.Fatal("expected cost estimated")
	}
	if e.CostUSD <= 0 {
		t.Fatalf("cost=%v", e.CostUSD)
	}
	if e.ModelAlias != "text-fast" {
		t.Fatalf("alias=%q", e.ModelAlias)
	}
}

func TestEntryFromCallMeta_preservesProviderCost(t *testing.T) {
	t.Parallel()
	meta := aiprovider.CallMeta{Provider: aiprovider.ProviderAnthropic, ModelID: "claude-3-5-sonnet-20241022"}
	usage := aiprovider.UsageInfo{PromptTokens: 10, CompletionTokens: 5, CostUSD: 0.02}
	e := EntryFromCallMeta(nil, nil, "grader_agent", meta, usage, true)
	if e.CostEstimated {
		t.Fatal("should not mark estimated")
	}
	if e.CostUSD != 0.02 {
		t.Fatalf("cost=%v", e.CostUSD)
	}
	if e.Provider != "anthropic" {
		t.Fatalf("provider=%q", e.Provider)
	}
}
