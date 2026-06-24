package gradingagent

import (
	"fmt"
	"strings"

	"github.com/lextures/lextures/server/internal/models/assignmentrubric"
)

func isScoreAggregatorNodeType(nodeType string) bool {
	return nodeType == NodeTypeScoreAggregator
}

func aggregatorInputSourceIsValid(src WorkflowNode, srcHandle string) bool {
	if srcHandle == HandleGrade &&
		(src.Type == NodeTypeGrader || isCriterionGraderNodeType(src.Type) || isCodeTestRunnerNodeType(src.Type) || isHumanReviewGateNodeType(src.Type)) {
		return true
	}
	if isAINodeType(src.Type) && srcHandle == HandleAIOutput {
		return true
	}
	if isConditionalRouterNodeType(src.Type) && (srcHandle == HandleThen || srcHandle == HandleElse) {
		return true
	}
	return false
}

func aggregatorModeFromNode(n WorkflowNode) AggregatorMode {
	if n.Data == nil {
		return AggregatorModeSum
	}
	if v, ok := n.Data["mode"].(string); ok {
		switch AggregatorMode(strings.TrimSpace(v)) {
		case AggregatorModeWeightedSum, AggregatorModeAverage, AggregatorModeMin, AggregatorModeMax, AggregatorModeRubricMerge:
			return AggregatorMode(strings.TrimSpace(v))
		}
	}
	return AggregatorModeSum
}

func aggregatorConfidenceFromNode(n WorkflowNode) AggregatorConfidenceMode {
	if n.Data == nil {
		return AggregatorConfidenceMin
	}
	if v, ok := n.Data["confidence"].(string); ok {
		switch AggregatorConfidenceMode(strings.TrimSpace(v)) {
		case AggregatorConfidenceMean, AggregatorConfidenceWeighted:
			return AggregatorConfidenceMode(strings.TrimSpace(v))
		}
	}
	return AggregatorConfidenceMin
}

func aggregatorOnMissingFromNode(n WorkflowNode) AggregatorOnMissing {
	if n.Data == nil {
		return AggregatorOnMissingTreatAsZero
	}
	if v, ok := n.Data["onMissing"].(string); ok {
		switch AggregatorOnMissing(strings.TrimSpace(v)) {
		case AggregatorOnMissingSkipAndRenormalize, AggregatorOnMissingFailItem:
			return AggregatorOnMissing(strings.TrimSpace(v))
		}
	}
	return AggregatorOnMissingTreatAsZero
}

func aggregatorMergeCommentsFromNode(n WorkflowNode) bool {
	if n.Data == nil {
		return true
	}
	v, ok := n.Data["mergeComments"].(bool)
	if !ok {
		return true
	}
	return v
}

func aggregatorWeightFromNode(n WorkflowNode, sourceID string) float64 {
	if n.Data == nil {
		return 1
	}
	raw, ok := n.Data["weights"].(map[string]any)
	if !ok {
		return 1
	}
	v, ok := raw[sourceID]
	if !ok {
		return 1
	}
	switch w := v.(type) {
	case float64:
		if w > 0 {
			return w
		}
	case int:
		if w > 0 {
			return float64(w)
		}
	}
	return 1
}

func aggregatorConfigFromNode(n WorkflowNode) AggregatorConfig {
	return AggregatorConfig{
		Mode:          aggregatorModeFromNode(n),
		Confidence:    aggregatorConfidenceFromNode(n),
		OnMissing:     aggregatorOnMissingFromNode(n),
		MergeComments: aggregatorMergeCommentsFromNode(n),
		CommentSep:    "\n\n",
	}
}

func aggregatorHasGradeInput(g *WorkflowGraph, nodeID string) bool {
	for _, e := range g.Edges {
		if e.Target == nodeID && strings.TrimSpace(e.TargetHandle) == HandleGrade {
			return true
		}
	}
	return false
}

func wiredAggregatorSourceCriterionIDs(g *WorkflowGraph, nodeID string, nodeByID map[string]WorkflowNode) []string {
	ids := make([]string, 0)
	for _, e := range g.Edges {
		if e.Target != nodeID || strings.TrimSpace(e.TargetHandle) != HandleGrade {
			continue
		}
		src, ok := nodeByID[e.Source]
		if !ok || !isCriterionGraderNodeType(src.Type) {
			continue
		}
		if src.Data == nil {
			continue
		}
		if v, ok := src.Data["criterionId"].(string); ok && strings.TrimSpace(v) != "" {
			ids = append(ids, strings.TrimSpace(v))
		}
	}
	return ids
}

func gatherAggregatorInputs(
	g *WorkflowGraph,
	node WorkflowNode,
	nodeByID map[string]WorkflowNode,
	state *executionState,
) ([]AggregatorInput, error) {
	cfg := aggregatorConfigFromNode(node)
	inputs := make([]AggregatorInput, 0)
	for _, e := range g.Edges {
		if e.Target != node.ID || strings.TrimSpace(e.TargetHandle) != HandleGrade {
			continue
		}
		if !state.edgeActive[e.ID] {
			continue
		}
		src, ok := nodeByID[e.Source]
		if !ok {
			continue
		}
		srcHandle := strings.TrimSpace(e.SourceHandle)
		label := NodeDisplayLabel(src.Data, src.Type)
		in := AggregatorInput{
			SourceID: src.ID,
			Label:    label,
			Weight:   aggregatorWeightFromNode(node, src.ID),
			Missing:  true,
		}
		if v, ok := state.get(src.ID, srcHandle); ok && v.grade != nil {
			copyGrade := *v.grade
			in.Grade = &copyGrade
			in.Missing = false
		}
		inputs = append(inputs, in)
	}
	if len(inputs) == 0 {
		return nil, ValidationError{Field: "node:" + node.ID + ".grade", Message: "Connect at least one grade input to the Score Aggregator."}
	}
	_ = cfg
	return inputs, nil
}

func executeScoreAggregatorNode(
	g *WorkflowGraph,
	node WorkflowNode,
	nodeByID map[string]WorkflowNode,
	state *executionState,
	maxPoints float64,
	rubric *assignmentrubric.RubricDefinition,
	emit func(DryRunEvent),
	label string,
) error {
	inputs, err := gatherAggregatorInputs(g, node, nodeByID, state)
	if err != nil {
		return err
	}
	cfg := aggregatorConfigFromNode(node)
	for i := range inputs {
		inputs[i].Weight = aggregatorWeightFromNode(node, inputs[i].SourceID)
	}
	grade, logs, combineErr := CombineGrades(inputs, cfg, maxPoints, rubric)
	if combineErr != nil {
		return combineErr
	}
	emit(DryRunEvent{
		Type: "log", Level: "info",
		Message: fmt.Sprintf("[%s] Aggregating %d input(s) via %s:", label, len(inputs), cfg.Mode),
	})
	for _, line := range logs {
		emit(DryRunEvent{Type: "log", Level: "info", Message: line})
	}
	state.set(node.ID, HandleGrade, slotValue{grade: &grade})
	if cfg.MergeComments && strings.TrimSpace(grade.Comment) != "" {
		state.set(node.ID, HandleComments, slotValue{text: grade.Comment})
	}
	return nil
}