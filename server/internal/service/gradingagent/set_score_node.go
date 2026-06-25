package gradingagent

import (
	"fmt"
	"strings"
)

func isSetScoreNodeType(nodeType string) bool {
	return nodeType == NodeTypeSetScore
}

func setScoreValueFromNode(n WorkflowNode) (float64, error) {
	if n.Data == nil {
		return 0, nil
	}
	switch v := n.Data["score"].(type) {
	case float64:
		return v, nil
	case int:
		return float64(v), nil
	case nil:
		return 0, nil
	default:
		return 0, fmt.Errorf("score must be a number")
	}
}

func setScoreCommentFromNode(n WorkflowNode) string {
	if n.Data == nil {
		return ""
	}
	if v, ok := n.Data["comment"].(string); ok {
		return strings.TrimSpace(v)
	}
	return ""
}

func setScoreInputSourceIsValid(src WorkflowNode, srcHandle string) bool {
	if isConditionalRouterNodeType(src.Type) && (srcHandle == HandleThen || srcHandle == HandleElse) {
		return true
	}
	if (src.Type == NodeTypeGrader || isCriterionGraderNodeType(src.Type)) && srcHandle == HandleGrade {
		return true
	}
	if isAINodeType(src.Type) && srcHandle == HandleAIOutput {
		return true
	}
	if isCodeTestRunnerNodeType(src.Type) && srcHandle == HandleGrade {
		return true
	}
	if isHumanReviewGateNodeType(src.Type) && srcHandle == HandleGrade {
		return true
	}
	if isScoreAggregatorNodeType(src.Type) && srcHandle == HandleGrade {
		return true
	}
	return false
}

func executeSetScoreNode(node WorkflowNode, state *executionState) error {
	score, err := setScoreValueFromNode(node)
	if err != nil {
		return err
	}
	comment := setScoreCommentFromNode(node)
	grade := &GradeOutput{
		TotalPoints: score,
		Confidence:  1,
		Comment:     comment,
	}
	state.set(node.ID, HandleGrade, slotValue{grade: grade})
	return nil
}
