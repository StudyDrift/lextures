package gradingagent

import (
	"testing"
)

func TestEvaluateRouterCondition_isEmpty(t *testing.T) {
	cond := RouterCondition{Field: "isEmpty", Operator: "isTrue", Value: true}
	got, err := EvaluateRouterCondition(cond, PredicateEvalContext{SubmissionText: "   "})
	if err != nil {
		t.Fatal(err)
	}
	if !got {
		t.Fatal("expected empty submission to match isEmpty")
	}
}

func TestEvaluateRouterCondition_wordCount(t *testing.T) {
	cond := RouterCondition{Field: "wordCount", Operator: "<", Value: 5.0}
	got, err := EvaluateRouterCondition(cond, PredicateEvalContext{SubmissionText: "one two three"})
	if err != nil {
		t.Fatal(err)
	}
	if !got {
		t.Fatal("expected wordCount < 5")
	}
}

func TestEvaluateRouterCondition_confidenceRequiresGrade(t *testing.T) {
	cond := RouterCondition{Field: "confidence", Operator: "<", Value: 0.6}
	_, err := EvaluateRouterCondition(cond, PredicateEvalContext{})
	if err == nil {
		t.Fatal("expected error without upstream grade")
	}
}

func TestEvaluateRouterCondition_confidence(t *testing.T) {
	grade := GradeOutput{Confidence: 0.4, TotalPoints: 8}
	cond := RouterCondition{Field: "confidence", Operator: "<", Value: 0.6}
	got, err := EvaluateRouterCondition(cond, PredicateEvalContext{InputGrade: &grade})
	if err != nil {
		t.Fatal(err)
	}
	if !got {
		t.Fatal("expected low confidence match")
	}
}

func TestEvaluateRouterCondition_contains(t *testing.T) {
	cond := RouterCondition{Field: "submissionText", Operator: "contains", Value: "hello"}
	got, err := EvaluateRouterCondition(cond, PredicateEvalContext{SubmissionText: "say hello world"})
	if err != nil {
		t.Fatal(err)
	}
	if !got {
		t.Fatal("expected contains match")
	}
}

func TestValidateRouterPathReachability_deadEnd(t *testing.T) {
	g := WorkflowGraph{
		Version: WorkflowVersion,
		Nodes: []WorkflowNode{
			{ID: "output", Type: NodeTypeOutput},
			{ID: "sub", Type: NodeTypeStudentSubmission},
			{ID: "r1", Type: NodeTypeConditionalRouter, Data: map[string]any{
				"condition": map[string]any{"field": "isEmpty", "operator": "isTrue", "value": true},
			}},
			{ID: "ai1", Type: NodeTypeAI, Data: map[string]any{"prompt": "Grade"}},
		},
		Edges: []WorkflowEdge{
			{ID: "e1", Source: "sub", SourceHandle: HandleSubmission, Target: "r1", TargetHandle: HandleAIInput},
			{ID: "e2", Source: "r1", SourceHandle: HandleThen, Target: "output", TargetHandle: HandleGrade},
			{ID: "e3", Source: "r1", SourceHandle: HandleElse, Target: "ai1", TargetHandle: HandleAIInput},
		},
	}
	nodeByID := make(map[string]WorkflowNode, len(g.Nodes))
	for _, n := range g.Nodes {
		nodeByID[n.ID] = n
	}
	err := validateRouterPathReachability(&g, nodeByID)
	if err == nil {
		t.Fatal("expected else branch reachability error")
	}
}

func TestValidateRouterFieldAvailability_confidenceWithoutGrade(t *testing.T) {
	g := WorkflowGraph{
		Version: WorkflowVersion,
		Nodes: []WorkflowNode{
			{ID: "output", Type: NodeTypeOutput},
			{ID: "sub", Type: NodeTypeStudentSubmission},
			{ID: "r1", Type: NodeTypeConditionalRouter, Data: map[string]any{
				"condition": map[string]any{"field": "confidence", "operator": "<", "value": 0.6},
			}},
		},
		Edges: []WorkflowEdge{
			{ID: "e1", Source: "sub", SourceHandle: HandleSubmission, Target: "r1", TargetHandle: HandleAIInput},
			{ID: "e2", Source: "r1", SourceHandle: HandleThen, Target: "output", TargetHandle: HandleGrade},
		},
	}
	nodeByID := make(map[string]WorkflowNode, len(g.Nodes))
	for _, n := range g.Nodes {
		nodeByID[n.ID] = n
	}
	err := validateRouterFieldAvailability(&g, nodeByID)
	if err == nil {
		t.Fatal("expected unavailable confidence field error")
	}
}

func sampleGraphWithRouterEmptyShortCircuit() WorkflowGraph {
	return WorkflowGraph{
		Version: WorkflowVersion,
		Nodes: []WorkflowNode{
			{ID: "output", Type: NodeTypeOutput},
			{ID: "sub", Type: NodeTypeStudentSubmission},
			{ID: "r1", Type: NodeTypeConditionalRouter, Data: map[string]any{
				"condition": map[string]any{"field": "isEmpty", "operator": "isTrue", "value": true},
			}},
			{ID: "ai1", Type: NodeTypeAI, Data: map[string]any{"prompt": "Grade the submission"}},
		},
		Edges: []WorkflowEdge{
			{ID: "e1", Source: "sub", SourceHandle: HandleSubmission, Target: "r1", TargetHandle: HandleAIInput},
			{ID: "e2", Source: "r1", SourceHandle: HandleThen, Target: "output", TargetHandle: HandleGrade},
			{ID: "e3", Source: "r1", SourceHandle: HandleElse, Target: "ai1", TargetHandle: HandleAIInput},
			{ID: "e4", Source: "ai1", SourceHandle: HandleAIOutput, Target: "output", TargetHandle: HandleGrade},
		},
	}
}

func TestExecuteWorkflowDryRun_routerSkipsAIBranchOnEmpty(t *testing.T) {
	g := sampleGraphWithRouterEmptyShortCircuit()
	if err := ValidateWorkflowGraph(&g); err != nil {
		t.Fatalf("graph should validate: %v", err)
	}
	var aiComplete string
	var skipped []string
	preview, err := ExecuteWorkflowDryRun(t.Context(), DryRunExecutionInput{
		Graph:       &g,
		Submissions: []string{""},
		MaxPoints:   10,
		ModelID:     "test/model",
		Runner:      stubDryRunRunner{},
		CodeRunner:  nil,
		Emit: func(ev DryRunEvent) {
			if ev.Type == "node_complete" && ev.NodeID == "ai1" {
				aiComplete = ev.Status
			}
			if ev.Type == "node_complete" && ev.Status == "skipped" {
				skipped = append(skipped, ev.NodeID)
			}
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if aiComplete != "skipped" {
		t.Fatalf("expected AI skipped, got %q", aiComplete)
	}
	if preview.SuggestedPoints != 0 {
		t.Fatalf("expected zero grade for empty submission, got %v", preview.SuggestedPoints)
	}
}
