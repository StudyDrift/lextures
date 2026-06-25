package gradingagent

import "testing"

func TestEvaluateHoldDecision_always(t *testing.T) {
	wouldHold, reason := EvaluateHoldDecision(GateModeAlways, 0.7, &GradeOutput{Confidence: 0.9}, false)
	if !wouldHold || reason == "" {
		t.Fatalf("always mode: wouldHold=%v reason=%q", wouldHold, reason)
	}
}

func TestEvaluateHoldDecision_belowConfidence(t *testing.T) {
	low := &GradeOutput{Confidence: 0.5}
	wouldHold, _ := EvaluateHoldDecision(GateModeBelowConfidence, 0.7, low, false)
	if !wouldHold {
		t.Fatal("expected hold for low confidence")
	}
	high := &GradeOutput{Confidence: 0.9}
	wouldHold, _ = EvaluateHoldDecision(GateModeBelowConfidence, 0.7, high, false)
	if wouldHold {
		t.Fatal("expected pass for high confidence")
	}
}

func TestEvaluateAgentConfidenceFloorHold(t *testing.T) {
	var floor float64 = 0.8
	wouldHold, reason := EvaluateAgentConfidenceFloorHold(&floor, 0.62)
	if !wouldHold || reason != "Confidence 0.62 < floor 0.80" {
		t.Fatalf("below floor: wouldHold=%v reason=%q", wouldHold, reason)
	}
	wouldHold, _ = EvaluateAgentConfidenceFloorHold(&floor, 0.91)
	if wouldHold {
		t.Fatal("expected pass above floor")
	}
	wouldHold, _ = EvaluateAgentConfidenceFloorHold(nil, 0.5)
	if wouldHold {
		t.Fatal("nil floor should not hold")
	}
	zero := 0.0
	wouldHold, _ = EvaluateAgentConfidenceFloorHold(&zero, 0.5)
	if wouldHold {
		t.Fatal("zero floor should not hold")
	}
}

func TestComposeHoldDecisions(t *testing.T) {
	wouldHold, reason := ComposeHoldDecisions(true, "Human review gate (always)", false, "")
	if !wouldHold || reason != "Human review gate (always)" {
		t.Fatalf("gate only: wouldHold=%v reason=%q", wouldHold, reason)
	}
	wouldHold, reason = ComposeHoldDecisions(false, "", true, "Confidence 0.62 < floor 0.80")
	if !wouldHold || reason != "Confidence 0.62 < floor 0.80" {
		t.Fatalf("floor only: wouldHold=%v reason=%q", wouldHold, reason)
	}
	wouldHold, reason = ComposeHoldDecisions(
		true, "Confidence 0.85 below floor 0.90",
		true, "Confidence 0.85 < floor 0.80",
	)
	if !wouldHold || reason != "Confidence 0.85 below floor 0.90; Confidence 0.85 < floor 0.80" {
		t.Fatalf("both: wouldHold=%v reason=%q", wouldHold, reason)
	}
	wouldHold, _ = ComposeHoldDecisions(false, "", false, "")
	if wouldHold {
		t.Fatal("expected no hold")
	}
}

func TestEvaluateHoldDecision_onFlag(t *testing.T) {
	grade := &GradeOutput{Confidence: 0.95}
	wouldHold, _ := EvaluateHoldDecision(GateModeOnFlag, 0.7, grade, true)
	if !wouldHold {
		t.Fatal("expected hold when flag is true")
	}
	wouldHold, _ = EvaluateHoldDecision(GateModeOnFlag, 0.7, grade, false)
	if wouldHold {
		t.Fatal("expected pass when flag is false")
	}
}