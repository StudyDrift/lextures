package httpserver

import (
	"testing"

	gradingagentrepo "github.com/lextures/lextures/server/internal/repos/gradingagent"
	gradingagentsvc "github.com/lextures/lextures/server/internal/service/gradingagent"
)

func TestGradingAgentHoldDecision_composesGateAndFloor(t *testing.T) {
	floor := 0.8
	cfg := &gradingagentrepo.ConfigRow{ConfidenceFloor: &floor}
	gate := &gradingAgentGateHoldInput{WouldHold: true, Reason: "Human review gate (always)", Queue: "instructor"}

	hold, reason, queue := gradingAgentHoldDecision(cfg, 0.95, gate)
	if !hold || queue != "instructor" {
		t.Fatalf("gate hold: hold=%v queue=%q", hold, queue)
	}
	if reason == "" {
		t.Fatal("expected hold reason")
	}

	hold, reason, _ = gradingAgentHoldDecision(cfg, 0.62, &gradingAgentGateHoldInput{})
	if !hold || reason == "" {
		t.Fatalf("floor hold: hold=%v reason=%q", hold, reason)
	}

	hold, _, _ = gradingAgentHoldDecision(cfg, 0.95, &gradingAgentGateHoldInput{})
	if hold {
		t.Fatal("expected no hold above floor without gate")
	}
}

func TestGradingAgentPreviewResult_routesToPersistInputs(t *testing.T) {
	preview := gradingAgentPreviewResult{
		Points: 9, Comment: "Nice", Confidence: 0.88, GradedByAI: true,
		RubricScores: map[string]float64{"c1": 4},
		Held: &gradingagentsvc.DryRunHeldPreview{
			WouldHold: true,
			Reason:    "Human review gate (below confidence)",
			Queue:     "default",
		},
	}
	gateHold := &gradingAgentGateHoldInput{}
	if preview.Held != nil {
		gateHold.WouldHold = preview.Held.WouldHold
		gateHold.Reason = preview.Held.Reason
		gateHold.Queue = preview.Held.Queue
	}
	hold, _, _ := gradingAgentHoldDecision(&gradingagentrepo.ConfigRow{}, preview.Confidence, gateHold)
	if !hold {
		t.Fatal("expected persist path to honor gate hold from preview")
	}
}
