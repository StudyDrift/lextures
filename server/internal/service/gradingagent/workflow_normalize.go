package gradingagent

import (
	"log/slog"
	"strings"
)

// NormalizeWorkflowGraph rewrites legacy node types and handles to their canonical forms.
// It is idempotent and safe to call on already-normalized graphs.
func NormalizeWorkflowGraph(g *WorkflowGraph) (WorkflowGraph, int) {
	if g == nil {
		return WorkflowGraph{}, 0
	}
	changes := 0
	out := *g
	out.Nodes = make([]WorkflowNode, len(g.Nodes))
	for i, n := range g.Nodes {
		node := n
		switch node.Type {
		case NodeTypeSubmission:
			node.Type = NodeTypeStudentSubmission
			changes++
		case NodeTypeAssignmentCtx:
			node.Type = NodeTypeActivity
			changes++
		}
		out.Nodes[i] = node
	}
	nodeByID := make(map[string]WorkflowNode, len(out.Nodes))
	for _, n := range out.Nodes {
		nodeByID[n.ID] = n
	}
	normalizedEdges, edgeChanges := normalizeWorkflowEdges(g.Edges, nodeByID)
	changes += edgeChanges
	out.Edges = normalizedEdges
	if changes > 0 {
		slog.Info("grading agent workflow graph normalized", "changes", changes)
	}
	return out, changes
}

func normalizeWorkflowEdges(edges []WorkflowEdge, nodeByID map[string]WorkflowNode) ([]WorkflowEdge, int) {
	if len(edges) == 0 {
		return nil, 0
	}
	changes := 0
	out := make([]WorkflowEdge, 0, len(edges))
	for _, e := range edges {
		src, srcOK := nodeByID[e.Source]
		tgt, tgtOK := nodeByID[e.Target]
		if !srcOK || !tgtOK {
			out = append(out, e)
			continue
		}
		targetHandle := strings.TrimSpace(e.TargetHandle)

		if targetHandle == HandleContext && isPromptConsumerNodeType(tgt.Type) {
			expanded, expandedChanges := expandContextEdge(e, src, tgt.Type)
			out = append(out, expanded...)
			changes += expandedChanges
			continue
		}

		edge := e
		if tgt.Type == NodeTypeAI && isPromptConsumerLegacyTargetHandle(targetHandle) {
			edge.TargetHandle = HandleAIInput
			changes++
		}
		out = append(out, edge)
	}
	return out, changes
}

func isPromptConsumerNodeType(nodeType string) bool {
	return nodeType == NodeTypeGrader || nodeType == NodeTypeCriterionGrader || nodeType == NodeTypeAI
}

func isPromptConsumerLegacyTargetHandle(targetHandle string) bool {
	switch targetHandle {
	case HandleSubmission, HandleContent, HandleRubric, HandleContext:
		return true
	default:
		return false
	}
}

func expandContextEdge(e WorkflowEdge, src WorkflowNode, targetNodeType string) ([]WorkflowEdge, int) {
	includeContent, includeRubric := activityContextIncludeFlags(src)
	if !includeContent && !includeRubric {
		return nil, 1
	}
	changes := 1
	out := make([]WorkflowEdge, 0, 2)
	if includeContent {
		out = append(out, WorkflowEdge{
			ID:           e.ID + "-content",
			Source:       e.Source,
			SourceHandle: HandleContent,
			Target:       e.Target,
			TargetHandle: contentTargetHandleForNode(targetNodeType),
		})
	}
	if includeRubric {
		out = append(out, WorkflowEdge{
			ID:           e.ID + "-rubric",
			Source:       e.Source,
			SourceHandle: HandleRubric,
			Target:       e.Target,
			TargetHandle: rubricTargetHandleForNode(targetNodeType),
		})
	}
	return out, changes
}

func contentTargetHandleForNode(nodeType string) string {
	if nodeType == NodeTypeAI {
		return HandleAIInput
	}
	return HandleContent
}

func rubricTargetHandleForNode(nodeType string) string {
	if nodeType == NodeTypeAI {
		return HandleAIInput
	}
	return HandleRubric
}

func activityContextIncludeFlags(src WorkflowNode) (includeContent, includeRubric bool) {
	if src.Type != NodeTypeActivity {
		return false, false
	}
	if _, hadIncludeContent := src.Data["includeContent"]; hadIncludeContent {
		return boolData(src, "includeContent"), boolData(src, "includeRubric")
	}
	return true, true
}
