package aiprovider

import (
	"strings"
)

// Price table for best-effort USD estimates when a provider omits usage.cost (AP.6 FR-2).
//
// # Update cadence
//
// Review rates at least quarterly against provider price pages / invoices.
// Prefer provider-reported cost when present; never invent rates for image models.
// Rates are USD per 1M tokens (input / output). Last reviewed: 2026-07.

type tokenRates struct {
	InputPerMillion  float64
	OutputPerMillion float64
}

// modelPrices is keyed by normalized "provider|model" (lowercase).
// Prefer native provider ids; OpenRouter slash ids are also listed for direct OpenRouter calls.
var modelPrices = map[string]tokenRates{
	// OpenAI / Azure OpenAI
	"openai|gpt-4o-mini":       {InputPerMillion: 0.15, OutputPerMillion: 0.60},
	"openai|gpt-4o":            {InputPerMillion: 2.50, OutputPerMillion: 10.00},
	"azure_openai|gpt-4o-mini": {InputPerMillion: 0.15, OutputPerMillion: 0.60},
	"azure_openai|gpt-4o":      {InputPerMillion: 2.50, OutputPerMillion: 10.00},

	// Anthropic
	"anthropic|claude-3-5-haiku-20241022":  {InputPerMillion: 0.80, OutputPerMillion: 4.00},
	"anthropic|claude-3-5-sonnet-20241022": {InputPerMillion: 3.00, OutputPerMillion: 15.00},
	"anthropic|claude-3-5-sonnet":         {InputPerMillion: 3.00, OutputPerMillion: 15.00},

	// Bedrock (Anthropic on AWS)
	"bedrock|anthropic.claude-3-5-haiku-20241022-v1:0":   {InputPerMillion: 0.80, OutputPerMillion: 4.00},
	"bedrock|anthropic.claude-3-5-sonnet-20241022-v2:0": {InputPerMillion: 3.00, OutputPerMillion: 15.00},

	// Vertex
	"vertex|gemini-1.5-flash": {InputPerMillion: 0.075, OutputPerMillion: 0.30},
	"vertex|gemini-1.5-pro":   {InputPerMillion: 1.25, OutputPerMillion: 5.00},

	// OpenRouter (common routed ids)
	"openrouter|openai/gpt-4o-mini":           {InputPerMillion: 0.15, OutputPerMillion: 0.60},
	"openrouter|openai/gpt-4o":                {InputPerMillion: 2.50, OutputPerMillion: 10.00},
	"openrouter|anthropic/claude-3.5-sonnet":  {InputPerMillion: 3.00, OutputPerMillion: 15.00},
	"openrouter|arcee-ai/trinity-mini:free":   {InputPerMillion: 0, OutputPerMillion: 0},
}

// EstimateCostUSD returns a best-effort USD estimate from the local price table.
// ok is false when rates are missing or token counts are zero (leave cost 0, not estimated).
func EstimateCostUSD(provider ProviderName, model string, promptTokens, completionTokens int) (cost float64, ok bool) {
	if promptTokens <= 0 && completionTokens <= 0 {
		return 0, false
	}
	rates, found := lookupRates(provider, model)
	if !found {
		return 0, false
	}
	if rates.InputPerMillion == 0 && rates.OutputPerMillion == 0 {
		return 0, true // known free tier
	}
	cost = (float64(promptTokens)/1_000_000)*rates.InputPerMillion +
		(float64(completionTokens)/1_000_000)*rates.OutputPerMillion
	return cost, true
}

// ApplyCostEstimate fills UsageInfo.CostUSD from the price table when the provider omitted cost.
// Returns whether cost_usd was estimated.
func ApplyCostEstimate(provider ProviderName, model string, usage *UsageInfo) bool {
	if usage == nil || usage.CostUSD > 0 {
		return false
	}
	est, ok := EstimateCostUSD(provider, model, usage.PromptTokens, usage.CompletionTokens)
	if !ok {
		return false
	}
	usage.CostUSD = est
	return true
}

func lookupRates(provider ProviderName, model string) (tokenRates, bool) {
	p := strings.ToLower(strings.TrimSpace(string(provider)))
	m := strings.ToLower(strings.TrimSpace(model))
	if p == "" || m == "" {
		return tokenRates{}, false
	}
	if rates, ok := modelPrices[p+"|"+m]; ok {
		return rates, true
	}
	// Strip OpenRouter :suffix variants (e.g. :free) already exact-matched above;
	// try without trailing :tag for paid sibling lookups.
	if i := strings.LastIndex(m, ":"); i > 0 {
		if rates, ok := modelPrices[p+"|"+m[:i]]; ok {
			return rates, true
		}
	}
	return tokenRates{}, false
}
