package gradingagent

import "strings"

func isHumanReviewGateNodeType(nodeType string) bool {
	return nodeType == NodeTypeHumanReviewGate
}

func gateInputSourceIsValid(src WorkflowNode, srcHandle, tgtHandle string) bool {
	if flagSinkInputSourceIsValid(src, srcHandle, tgtHandle) {
		return true
	}
	if tgtHandle == HandleFlag && isOriginalityNodeType(src.Type) && srcHandle == HandleFlag {
		return true
	}
	return false
}

func gatherGateGrade(
	g *WorkflowGraph,
	gateNodeID string,
	nodeByID map[string]WorkflowNode,
	state *executionState,
) (*GradeOutput, error) {
	for _, e := range g.Edges {
		if e.Target != gateNodeID || strings.TrimSpace(e.TargetHandle) != HandleGrade {
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
		v, ok := state.get(src.ID, srcHandle)
		if !ok {
			continue
		}
		if v.grade != nil {
			return v.grade, nil
		}
	}
	return nil, ValidationError{Field: "node:" + gateNodeID, Message: "Human Review Gate requires a grade input."}
}

func gatherGateFlagTruthy(
	g *WorkflowGraph,
	gateNodeID string,
	nodeByID map[string]WorkflowNode,
	state *executionState,
) bool {
	for _, e := range g.Edges {
		if e.Target != gateNodeID || strings.TrimSpace(e.TargetHandle) != HandleFlag {
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
		v, ok := state.get(src.ID, srcHandle)
		if !ok {
			continue
		}
		if v.flag != nil && *v.flag {
			return true
		}
		text := strings.TrimSpace(v.text)
		if text == "true" || text == "1" {
			return true
		}
	}
	return false
}

func gateHasGradeInput(g *WorkflowGraph, gateNodeID string) bool {
	for _, e := range g.Edges {
		if e.Target == gateNodeID && strings.TrimSpace(e.TargetHandle) == HandleGrade {
			return true
		}
	}
	return false
}

func resolveGradeWireSource(g *WorkflowGraph, nodeByID map[string]WorkflowNode, nodeID string) string {
	n, ok := nodeByID[nodeID]
	if !ok {
		return ""
	}
	if gradeSourceNodeType(n.Type) {
		return nodeID
	}
	if n.Type == NodeTypeHumanReviewGate {
		for _, e := range g.Edges {
			if e.Target != nodeID || strings.TrimSpace(e.TargetHandle) != HandleGrade {
				continue
			}
			if src := resolveGradeWireSource(g, nodeByID, e.Source); src != "" {
				return src
			}
		}
	}
	return ""
}