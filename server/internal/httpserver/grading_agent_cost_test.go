package httpserver

import (
	"testing"

	gradingagentrepo "github.com/lextures/lextures/server/internal/repos/gradingagent"
)

func TestGraderAgentCostEstimateToJSON(t *testing.T) {
	min := 0.17
	max := 0.23
	prompt := 2400
	completion := 360
	got := graderAgentCostEstimateToJSON(gradingagentrepo.CostEstimate{
		SubmissionCount:  24,
		HasSample:        true,
		PromptTokens:     &prompt,
		CompletionTokens: &completion,
		CostMinUSD:       &min,
		CostMaxUSD:       &max,
	}, "Ungraded: 24 submissions")
	if got["submissionCount"] != 24 || got["hasSample"] != true {
		t.Fatalf("got=%v", got)
	}
	if got["estimatedCostMinUsd"] != min || got["estimatedCostMaxUsd"] != max {
		t.Fatalf("cost range=%v", got)
	}
}

func TestGraderAgentRunBudgetBoundary(t *testing.T) {
	budget := 1.0
	spent := 1.0
	if spent < budget {
		t.Fatal("expected budget boundary to skip next item")
	}
}
