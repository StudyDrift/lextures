package gradingagent

import "math"

const costEstimateRangeLow = 0.85
const costEstimateRangeHigh = 1.15

// DryRunCostSample is token/cost usage from the most recent dry run for an agent config.
type DryRunCostSample struct {
	PromptTokens     *int
	CompletionTokens *int
	CostUSD          *float64
	ModelID          *string
}

// RunUsageTotals aggregates token and cost usage for a batch run (non-dry-run results).
type RunUsageTotals struct {
	PromptTokens     int
	CompletionTokens int
	CostUSD          float64
}

// CostEstimate projects batch spend from a per-submission sample and submission count.
type CostEstimate struct {
	SubmissionCount int
	HasSample       bool
	PromptTokens    *int
	CompletionTokens *int
	CostMinUSD      *float64
	CostMaxUSD      *float64
	TokensOnly      bool
}

// EstimateRunCost multiplies the dry-run sample by submission count and applies an approximate range.
func EstimateRunCost(count int, sample *DryRunCostSample) CostEstimate {
	out := CostEstimate{SubmissionCount: count}
	if count <= 0 {
		return out
	}
	if sample == nil {
		return out
	}
	out.HasSample = sample.PromptTokens != nil || sample.CompletionTokens != nil || sample.CostUSD != nil
	if sample.PromptTokens != nil {
		total := *sample.PromptTokens * count
		out.PromptTokens = &total
	}
	if sample.CompletionTokens != nil {
		total := *sample.CompletionTokens * count
		out.CompletionTokens = &total
	}
	if sample.CostUSD != nil && *sample.CostUSD > 0 {
		base := *sample.CostUSD * float64(count)
		min := roundCostUSD(base * costEstimateRangeLow)
		max := roundCostUSD(base * costEstimateRangeHigh)
		out.CostMinUSD = &min
		out.CostMaxUSD = &max
		return out
	}
	if out.HasSample {
		out.TokensOnly = true
	}
	return out
}

func roundCostUSD(v float64) float64 {
	return math.Round(v*1e4) / 1e4
}
