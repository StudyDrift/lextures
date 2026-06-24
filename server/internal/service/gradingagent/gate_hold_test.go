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