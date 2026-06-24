package gradingagent

import (
	"strings"
	"testing"
)

func sampleGraphWithReferenceAI() WorkflowGraph {
	return WorkflowGraph{
		Version: WorkflowVersion,
		Nodes: []WorkflowNode{
			{ID: "output", Type: NodeTypeOutput, Position: map[string]any{"x": 0, "y": 0}, Data: map[string]any{}},
			{ID: "ai1", Type: NodeTypeAI, Position: map[string]any{"x": -320, "y": 0}, Data: map[string]any{"prompt": "Grade using $ModelAnswer.Text"}},
			{ID: "ref1", Type: NodeTypeReference, Position: map[string]any{"x": -640, "y": 0}, Data: map[string]any{
				"mode":  "modelAnswer",
				"text":  "The ideal answer discusses supply and demand.",
				"label": "Model Answer",
			}},
			{ID: "sub1", Type: NodeTypeStudentSubmission, Position: map[string]any{"x": -640, "y": 120}, Data: map[string]any{}},
		},
		Edges: []WorkflowEdge{
			{ID: "e1", Source: "ai1", SourceHandle: HandleAIOutput, Target: "output", TargetHandle: HandleGrade},
			{ID: "e2", Source: "ref1", SourceHandle: HandleReference, Target: "ai1", TargetHandle: HandleAIInput},
			{ID: "e3", Source: "sub1", SourceHandle: HandleSubmission, Target: "ai1", TargetHandle: HandleAIInput},
		},
	}
}

func TestValidateWorkflowGraph_rejectsReferenceToGradeSlot(t *testing.T) {
	g := sampleGraphWithReferenceAI()
	g.Edges = append(g.Edges, WorkflowEdge{
		ID: "bad", Source: "ref1", SourceHandle: HandleReference, Target: "output", TargetHandle: HandleGrade,
	})
	err := ValidateWorkflowGraph(&g)
	if err == nil {
		t.Fatal("expected validation error for reference → grade")
	}
}

func TestValidateWorkflowGraph_requiresReferenceSource(t *testing.T) {
	g := sampleGraphWithReferenceAI()
	g.Nodes[2].Data = map[string]any{"mode": "modelAnswer", "text": ""}
	err := ValidateWorkflowGraph(&g)
	if err == nil {
		t.Fatal("expected validation error for empty reference")
	}
}

func TestFormatReferenceTrustedAIBlock_labelsByMode(t *testing.T) {
	node := WorkflowNode{Data: map[string]any{"mode": "answerKey"}}
	block := formatReferenceTrustedAIBlock(node, "x = 42")
	if !strings.Contains(block, "Answer Key (reference — trusted)") {
		t.Fatalf("unexpected block: %q", block)
	}
}

func TestExecuteWorkflowDryRun_referenceInAIInput(t *testing.T) {
	g := sampleGraphWithReferenceAI()
	var compiledInput string
	var compiledPrompt string
	_, err := ExecuteWorkflowDryRun(t.Context(), DryRunExecutionInput{
		Graph:       &g,
		Submissions: []string{"Student essay"},
		MaxPoints:   100,
		ModelID:     "test/model",
		Runner:      stubDryRunRunner{},
		Emit: func(ev DryRunEvent) {
			if ev.Type == "node_complete" && ev.NodeID == "ai1" {
				compiledInput = ev.CompiledInput
				compiledPrompt = ev.CompiledPrompt
			}
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(compiledInput, "Model Answer (reference — trusted)") {
		t.Fatalf("compiled input missing trusted reference block: %q", compiledInput)
	}
	if !strings.Contains(compiledInput, "supply and demand") {
		t.Fatalf("compiled input missing reference text: %q", compiledInput)
	}
	if !strings.Contains(compiledPrompt, "supply and demand") {
		t.Fatalf("compiled prompt missing substituted variable: %q", compiledPrompt)
	}
}

func TestSubstituteWorkflowPromptVariables_referenceText(t *testing.T) {
	g := sampleGraphWithReferenceAI()
	resolved := SubstituteWorkflowPromptVariables(&g, "ai1", "Key: $ModelAnswer.Text", PromptVariableContext{
		ReferenceTexts: map[string]string{"ref1": "Answer key body"},
	})
	if resolved != "Key: Answer key body" {
		t.Fatalf("resolved = %q", resolved)
	}
}