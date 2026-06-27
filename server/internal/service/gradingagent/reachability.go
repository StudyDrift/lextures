package gradingagent

import (
	"fmt"
	"strings"
)

// validateRouterPathReachability ensures every wired router branch reaches a terminal sink.
func validateRouterPathReachability(g *WorkflowGraph, nodeByID map[string]WorkflowNode) error {
	outputID := ""
	for _, n := range g.Nodes {
		if n.Type == NodeTypeOutput {
			outputID = n.ID
			break
		}
	}

	for _, n := range g.Nodes {
		if n.Type != NodeTypeConditionalRouter {
			continue
		}
		for _, handle := range []string{HandleThen, HandleElse} {
			if !routerHandleHasEdges(g, n.ID, handle) {
				continue
			}
			if !branchReachesTerminal(g, n.ID, handle, outputID, nodeByID) {
				label := handle
				return ValidationError{
					Field:   "node:" + n.ID + "." + handle,
					Message: fmt.Sprintf("The %s branch must reach a terminal (Student Grade or Flag for Review).", label),
				}
			}
		}
	}

	if graphHasRouter(g) && !anyPathReachesTerminal(g, outputID, nodeByID) {
		return ValidationError{Field: "workflowGraph.nodes", Message: "At least one executable path must reach a terminal."}
	}
	return nil
}

func graphHasRouter(g *WorkflowGraph) bool {
	for _, n := range g.Nodes {
		if n.Type == NodeTypeConditionalRouter {
			return true
		}
	}
	return false
}

func routerHandleHasEdges(g *WorkflowGraph, routerID, handle string) bool {
	for _, e := range g.Edges {
		if e.Source == routerID && strings.TrimSpace(e.SourceHandle) == handle {
			return true
		}
	}
	return false
}

func branchReachesTerminal(g *WorkflowGraph, routerID, handle, outputID string, nodeByID map[string]WorkflowNode) bool {
	for _, e := range g.Edges {
		if e.Source != routerID || strings.TrimSpace(e.SourceHandle) != handle {
			continue
		}
		tgt, ok := nodeByID[e.Target]
		if !ok {
			continue
		}
		if tgt.Type == NodeTypeFlagForReview {
			return true
		}
		if tgt.Type == NodeTypeOutput && outputID != "" && e.Target == outputID &&
			strings.TrimSpace(e.TargetHandle) == HandleGrade {
			return true
		}
	}
	starts := make([]string, 0)
	for _, e := range g.Edges {
		if e.Source == routerID && strings.TrimSpace(e.SourceHandle) == handle {
			starts = append(starts, e.Target)
		}
	}
	if len(starts) == 0 {
		return false
	}
	reachable := forwardReachable(g, starts)
	for id, node := range nodeByID {
		if node.Type == NodeTypeFlagForReview && reachable[id] {
			return true
		}
	}
	if outputID != "" {
		for _, e := range g.Edges {
			if e.Target != outputID || strings.TrimSpace(e.TargetHandle) != HandleGrade {
				continue
			}
			if reachable[e.Source] {
				return true
			}
		}
	}
	return false
}

func anyPathReachesTerminal(g *WorkflowGraph, outputID string, nodeByID map[string]WorkflowNode) bool {
	sources := make([]string, 0)
	for _, n := range g.Nodes {
		if isWorkflowSourceNode(n.Type) {
			sources = append(sources, n.ID)
		}
	}
	reachable := forwardReachable(g, sources)
	for id, node := range nodeByID {
		if node.Type == NodeTypeFlagForReview && reachable[id] {
			return true
		}
	}
	if outputID != "" {
		for _, e := range g.Edges {
			if e.Target != outputID || strings.TrimSpace(e.TargetHandle) != HandleGrade {
				continue
			}
			if reachable[e.Source] {
				return true
			}
		}
	}
	return false
}

func forwardReachable(g *WorkflowGraph, starts []string) map[string]bool {
	adj := make(map[string][]string)
	for _, e := range g.Edges {
		adj[e.Source] = append(adj[e.Source], e.Target)
	}
	seen := make(map[string]bool, len(starts))
	queue := append([]string(nil), starts...)
	for len(queue) > 0 {
		id := queue[0]
		queue = queue[1:]
		if seen[id] {
			continue
		}
		seen[id] = true
		queue = append(queue, adj[id]...)
	}
	return seen
}

func isWorkflowSourceNode(nodeType string) bool {
	switch nodeType {
	case NodeTypeStudentSubmission, NodeTypeActivity, NodeTypeReference, NodeTypeRubric:
		return true
	default:
		return false
	}
}

// validateRouterFieldAvailability checks that router conditions only use fields available upstream.
func validateRouterFieldAvailability(g *WorkflowGraph, nodeByID map[string]WorkflowNode) error {
	for _, n := range g.Nodes {
		if n.Type != NodeTypeConditionalRouter {
			continue
		}
		cond, err := routerConditionFromNode(n)
		if err != nil {
			return ValidationError{Field: "node:" + n.ID + ".condition", Message: err.Error()}
		}
		available := availableRouterFields(g, n.ID, nodeByID)
		if !available[cond.Field] {
			return ValidationError{
				Field:   "node:" + n.ID + ".condition.field",
				Message: fmt.Sprintf("Field %q is not available on this node's input path.", cond.Field),
			}
		}
	}
	return nil
}

func availableRouterFields(g *WorkflowGraph, routerID string, nodeByID map[string]WorkflowNode) map[string]bool {
	available := map[string]bool{
		"submissionLength": true,
		"wordCount":        true,
		"isEmpty":          true,
		"isLate":           true,
		"submissionText":   true,
		"matchesRegex":     true,
	}
	if upstreamProvidesGrade(g, routerID, nodeByID) {
		available["score"] = true
		available["confidence"] = true
	}
	if upstreamProvidesOriginality(g, routerID, nodeByID) {
		available["originalityScore"] = true
	}
	return available
}

func upstreamProvidesGrade(g *WorkflowGraph, routerID string, nodeByID map[string]WorkflowNode) bool {
	inputSources := routerInputSources(g, routerID)
	visited := make(map[string]bool)
	for _, srcID := range inputSources {
		if walkUpstreamForGrade(g, srcID, nodeByID, visited) {
			return true
		}
	}
	return false
}

func upstreamProvidesOriginality(g *WorkflowGraph, routerID string, nodeByID map[string]WorkflowNode) bool {
	inputSources := routerInputSources(g, routerID)
	visited := make(map[string]bool)
	for _, srcID := range inputSources {
		if walkUpstreamForOriginality(g, srcID, nodeByID, visited) {
			return true
		}
	}
	return false
}

func walkUpstreamForOriginality(g *WorkflowGraph, nodeID string, nodeByID map[string]WorkflowNode, visited map[string]bool) bool {
	if visited[nodeID] {
		return false
	}
	visited[nodeID] = true
	n, ok := nodeByID[nodeID]
	if !ok {
		return false
	}
	if n.Type == NodeTypeOriginality {
		return true
	}
	if n.Type == NodeTypeConditionalRouter {
		for _, srcID := range routerInputSources(g, nodeID) {
			if walkUpstreamForOriginality(g, srcID, nodeByID, visited) {
				return true
			}
		}
		return false
	}
	for _, e := range g.Edges {
		if e.Target != nodeID {
			continue
		}
		if walkUpstreamForOriginality(g, e.Source, nodeByID, visited) {
			return true
		}
	}
	return false
}

func routerInputSources(g *WorkflowGraph, routerID string) []string {
	out := make([]string, 0)
	for _, e := range g.Edges {
		if e.Target != routerID || strings.TrimSpace(e.TargetHandle) != HandleAIInput {
			continue
		}
		out = append(out, e.Source)
	}
	return out
}

func walkUpstreamForGrade(g *WorkflowGraph, nodeID string, nodeByID map[string]WorkflowNode, visited map[string]bool) bool {
	if visited[nodeID] {
		return false
	}
	visited[nodeID] = true
	n, ok := nodeByID[nodeID]
	if !ok {
		return false
	}
	switch n.Type {
	case NodeTypeGrader, NodeTypeCriterionGrader, NodeTypeAI, NodeTypeCodeTestRunner:
		return true
	case NodeTypeConditionalRouter:
		for _, srcID := range routerInputSources(g, nodeID) {
			if walkUpstreamForGrade(g, srcID, nodeByID, visited) {
				return true
			}
		}
		return false
	default:
		for _, e := range g.Edges {
			if e.Target != nodeID {
				continue
			}
			if walkUpstreamForGrade(g, e.Source, nodeByID, visited) {
				return true
			}
		}
		return false
	}
}

func isConditionalRouterNodeType(nodeType string) bool {
	return nodeType == NodeTypeConditionalRouter
}

func routerInputSourceIsValid(sourceType, sourceHandle string) bool {
	if isStudentSubmissionNodeType(sourceType) && sourceHandle == HandleSubmission {
		return true
	}
	if isQuizResponsesNodeType(sourceType) && isQuizQuestionHandle(sourceHandle) {
		return true
	}
	if isAINodeType(sourceType) && sourceHandle == HandleAIOutput {
		return true
	}
	if sourceType == NodeTypeGrader || sourceType == NodeTypeCriterionGrader {
		return sourceHandle == HandleGrade || sourceHandle == HandleComments
	}
	if isCodeTestRunnerNodeType(sourceType) {
		return sourceHandle == HandleGrade || sourceHandle == HandleReport || sourceHandle == HandleScore
	}
	if isConditionalRouterNodeType(sourceType) {
		return sourceHandle == HandleThen || sourceHandle == HandleElse
	}
	if isOriginalityNodeType(sourceType) {
		return sourceHandle == HandleScore || sourceHandle == HandleReport || sourceHandle == HandleFlag
	}
	return false
}