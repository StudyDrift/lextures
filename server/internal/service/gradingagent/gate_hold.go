package gradingagent

import (
	"fmt"
	"strings"
)

type HumanReviewGateMode string

const (
	GateModeAlways          HumanReviewGateMode = "always"
	GateModeBelowConfidence HumanReviewGateMode = "belowConfidence"
	GateModeOnFlag          HumanReviewGateMode = "onFlag"
)

func gateModeFromNode(n WorkflowNode) HumanReviewGateMode {
	if n.Data == nil {
		return GateModeBelowConfidence
	}
	if v, ok := n.Data["mode"].(string); ok {
		switch HumanReviewGateMode(strings.TrimSpace(v)) {
		case GateModeAlways, GateModeBelowConfidence, GateModeOnFlag:
			return HumanReviewGateMode(strings.TrimSpace(v))
		}
	}
	return GateModeBelowConfidence
}

func gateConfidenceFloorFromNode(n WorkflowNode) float64 {
	if n.Data == nil {
		return 0.7
	}
	switch v := n.Data["confidenceFloor"].(type) {
	case float64:
		if v >= 0 && v <= 1 {
			return v
		}
	case int:
		if v >= 0 && v <= 1 {
			return float64(v)
		}
	}
	return 0.7
}

func gateQueueFromNode(n WorkflowNode) string {
	if n.Data == nil {
		return "default"
	}
	if v, ok := n.Data["queue"].(string); ok && strings.TrimSpace(v) != "" {
		return strings.TrimSpace(v)
	}
	return "default"
}

// EvaluateHoldDecision reports whether a live run should hold the grade for human review.
func EvaluateHoldDecision(mode HumanReviewGateMode, floor float64, grade *GradeOutput, flagTruthy bool) (wouldHold bool, reason string) {
	switch mode {
	case GateModeAlways:
		return true, "Human review gate (always)"
	case GateModeBelowConfidence:
		if grade == nil {
			return true, "No grade confidence available"
		}
		if grade.Confidence < floor {
			return true, fmt.Sprintf("Confidence %.2f below floor %.2f", grade.Confidence, floor)
		}
		return false, ""
	case GateModeOnFlag:
		if flagTruthy {
			return true, "Upstream flag set"
		}
		return false, ""
	default:
		return false, ""
	}
}

// EvaluateAgentConfidenceFloorHold reports whether an agent-level confidence floor should hold a grade.
// A nil or non-positive floor is a no-op (preserves legacy behavior).
func EvaluateAgentConfidenceFloorHold(floor *float64, confidence float64) (wouldHold bool, reason string) {
	if floor == nil || *floor <= 0 {
		return false, ""
	}
	if confidence < *floor {
		return true, fmt.Sprintf("Confidence %.2f < floor %.2f", confidence, *floor)
	}
	return false, ""
}

// ComposeHoldDecisions merges gate and agent-level hold outcomes; either hold wins with combined reasons.
func ComposeHoldDecisions(hold1 bool, reason1 string, hold2 bool, reason2 string) (wouldHold bool, reason string) {
	if !hold1 && !hold2 {
		return false, ""
	}
	var parts []string
	if hold1 && strings.TrimSpace(reason1) != "" {
		parts = append(parts, strings.TrimSpace(reason1))
	}
	if hold2 && strings.TrimSpace(reason2) != "" {
		parts = append(parts, strings.TrimSpace(reason2))
	}
	if len(parts) == 0 {
		return true, "Held for human review"
	}
	return true, strings.Join(parts, "; ")
}