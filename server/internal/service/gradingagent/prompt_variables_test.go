package gradingagent

import (
	"strings"
	"testing"
)

func samplePromptVariableGraph() WorkflowGraph {
	return WorkflowGraph{
		Version: WorkflowVersion,
		Nodes: []WorkflowNode{
			{ID: "output", Type: NodeTypeOutput, Position: map[string]any{"x": 0, "y": 0}, Data: map[string]any{}},
			{ID: "ai1", Type: NodeTypeAI, Position: map[string]any{"x": -320, "y": 0}, Data: map[string]any{"prompt": "Grade"}},
			{ID: "sub1", Type: NodeTypeStudentSubmission, Position: map[string]any{"x": -640, "y": 0}, Data: map[string]any{}},
			{ID: "act1", Type: NodeTypeActivity, Position: map[string]any{"x": -640, "y": 120}, Data: map[string]any{"label": "Assignment Context"}},
		},
		Edges: []WorkflowEdge{
			{ID: "e1", Source: "sub1", SourceHandle: HandleSubmission, Target: "ai1", TargetHandle: HandleAIInput},
			{ID: "e2", Source: "act1", SourceHandle: HandleContent, Target: "ai1", TargetHandle: HandleAIInput},
			{ID: "e3", Source: "act1", SourceHandle: HandleRubric, Target: "ai1", TargetHandle: HandleAIInput},
		},
	}
}

func TestSubstituteWorkflowPromptVariables(t *testing.T) {
	g := samplePromptVariableGraph()
	prompt := strings.Join([]string{
		"Content: $AssignmentContext.Content",
		"Rubric: $AssignmentContext.Rubric",
		"Submission: $StudentSubmission.Submissions",
	}, "\n")
	resolved := SubstituteWorkflowPromptVariables(&g, "ai1", prompt, PromptVariableContext{
		Submissions: []string{"Student answer"},
		ContentMarkdown: "Essay prompt",
		Rubric:          nil,
	})
	if !strings.Contains(resolved, "Content: Essay prompt") {
		t.Fatalf("content not substituted: %q", resolved)
	}
	if !strings.Contains(resolved, "Submission: Student answer") {
		t.Fatalf("submission not substituted: %q", resolved)
	}
	if strings.Contains(resolved, "$AssignmentContext.Content") {
		t.Fatalf("expected content variable replaced: %q", resolved)
	}
}

func TestSubstitutePromptVariables_leavesUnknownTokens(t *testing.T) {
	prompt := "Missing: $Unknown.Value"
	resolved := SubstitutePromptVariables(prompt, map[string]map[string]string{
		"Activity": {"Content": "x"},
	})
	if resolved != prompt {
		t.Fatalf("expected unknown token preserved, got %q", resolved)
	}
}