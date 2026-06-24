package gradingagent

import "strings"

func isFlagForReviewNodeType(nodeType string) bool {
	return nodeType == NodeTypeFlagForReview
}

func graphHasFlagSink(g *WorkflowGraph) bool {
	for _, n := range g.Nodes {
		if n.Type == NodeTypeFlagForReview {
			return true
		}
	}
	return false
}

// WorkflowRequiresGraphExecution reports graphs that must be walked at runtime (routers / flag sinks).
func WorkflowRequiresGraphExecution(g *WorkflowGraph) bool {
	if g == nil {
		return false
	}
	for _, n := range g.Nodes {
		if n.Type == NodeTypeConditionalRouter || n.Type == NodeTypeFlagForReview || n.Type == NodeTypeHumanReviewGate {
			return true
		}
	}
	return false
}

func flagQueueFromNode(n WorkflowNode) string {
	if n.Data == nil {
		return "default"
	}
	if v, ok := n.Data["queue"].(string); ok && strings.TrimSpace(v) != "" {
		return strings.TrimSpace(v)
	}
	return "default"
}

func flagPriorityFromNode(n WorkflowNode) string {
	if n.Data == nil {
		return "normal"
	}
	if v, ok := n.Data["priority"].(string); ok && strings.TrimSpace(v) != "" {
		return strings.TrimSpace(v)
	}
	return "normal"
}

func flagReasonTemplateFromNode(n WorkflowNode) string {
	if n.Data == nil {
		return ""
	}
	if v, ok := n.Data["reasonTemplate"].(string); ok {
		return strings.TrimSpace(v)
	}
	return ""
}

func flagSinkInputSourceIsValid(src WorkflowNode, srcHandle, tgtHandle string) bool {
	switch tgtHandle {
	case HandleReason, HandleComments:
		if isStudentSubmissionNodeType(src.Type) && srcHandle == HandleSubmission {
			return true
		}
		if (src.Type == NodeTypeGrader || isCriterionGraderNodeType(src.Type)) &&
			(srcHandle == HandleComments || srcHandle == HandleGrade) {
			return true
		}
		if isAINodeType(src.Type) && srcHandle == HandleAIOutput {
			return true
		}
		if isCodeTestRunnerNodeType(src.Type) && (srcHandle == HandleReport || srcHandle == HandleGrade) {
			return true
		}
		if isConditionalRouterNodeType(src.Type) && (srcHandle == HandleThen || srcHandle == HandleElse) {
			return true
		}
		if isOriginalityNodeType(src.Type) && srcHandle == HandleReport {
			return true
		}
	case HandleReport:
		if isCodeTestRunnerNodeType(src.Type) && srcHandle == HandleReport {
			return true
		}
		if isConditionalRouterNodeType(src.Type) && (srcHandle == HandleThen || srcHandle == HandleElse) {
			return true
		}
		if isOriginalityNodeType(src.Type) && srcHandle == HandleReport {
			return true
		}
	case HandleGrade:
		if srcHandle == HandleGrade &&
			(src.Type == NodeTypeGrader || isCriterionGraderNodeType(src.Type) || isCodeTestRunnerNodeType(src.Type)) {
			return true
		}
		if srcHandle == HandleAIOutput && isAINodeType(src.Type) {
			return true
		}
		if isConditionalRouterNodeType(src.Type) && (srcHandle == HandleThen || srcHandle == HandleElse) {
			return true
		}
	case HandleFlag:
		if isConditionalRouterNodeType(src.Type) && (srcHandle == HandleThen || srcHandle == HandleElse) {
			return true
		}
		if isOriginalityNodeType(src.Type) && srcHandle == HandleFlag {
			return true
		}
	}
	return false
}

func gatherFlagSlotText(
	g *WorkflowGraph,
	flagNodeID, tgtHandle string,
	nodeByID map[string]WorkflowNode,
	state *executionState,
) string {
	var parts []string
	for _, e := range g.Edges {
		if e.Target != flagNodeID || strings.TrimSpace(e.TargetHandle) != tgtHandle {
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
		if v.grade != nil && strings.TrimSpace(v.grade.Comment) != "" {
			parts = append(parts, v.grade.Comment)
			continue
		}
		if strings.TrimSpace(v.text) != "" {
			parts = append(parts, v.text)
		}
	}
	return strings.TrimSpace(strings.Join(parts, "\n\n"))
}

func assembleFlagForReview(
	g *WorkflowGraph,
	node WorkflowNode,
	nodeByID map[string]WorkflowNode,
	state *executionState,
	in DryRunExecutionInput,
) (reason, queue, priority string) {
	queue = flagQueueFromNode(node)
	priority = flagPriorityFromNode(node)
	template := flagReasonTemplateFromNode(node)

	promptCtx := buildPromptContext(g, node.ID, nodeByID, state, in.Submissions, in.DefaultMarkdown, in.DefaultRubric)
	reason = SubstituteWorkflowPromptVariables(g, node.ID, template, promptCtx)

	for _, slot := range []string{HandleReason, HandleComments, HandleReport} {
		if wired := gatherFlagSlotText(g, node.ID, slot, nodeByID, state); wired != "" {
			if reason == "" {
				reason = wired
			} else if slot == HandleReason {
				reason = wired
			} else if !strings.Contains(reason, wired) {
				reason = strings.TrimSpace(reason + "\n" + wired)
			}
		}
	}
	if reason == "" {
		reason = "Flagged for human review"
	}
	return reason, queue, priority
}