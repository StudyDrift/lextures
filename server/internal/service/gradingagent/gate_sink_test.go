package gradingagent

import (
	"context"
	"strings"
	"testing"
)

func sampleGraphAIGateOutput() WorkflowGraph {
	return WorkflowGraph{
		Version: WorkflowVersion,
		Nodes: []WorkflowNode{
			{ID: "output", Type: NodeTypeOutput},
			{ID: "sub", Type: NodeTypeStudentSubmission},
			{ID: "ai1", Type: NodeTypeAI, Data: map[string]any{"prompt": "Grade the submission"}},
			{ID: "gate1", Type: NodeTypeHumanReviewGate, Data: map[string]any{
				"mode": "belowConfidence", "confidenceFloor": 0.7,
			}},
		},
		Edges: []WorkflowEdge{
			{ID: "e1", Source: "sub", SourceHandle: HandleSubmission, Target: "ai1", TargetHandle: HandleAIInput},
			{ID: "e2", Source: "ai1", SourceHandle: HandleAIOutput, Target: "gate1", TargetHandle: HandleGrade},
			{ID: "e3", Source: "gate1", SourceHandle: HandleGrade, Target: "output", TargetHandle: HandleGrade},
		},
	}
}

func TestValidateWorkflowGraph_aiGateOutput(t *testing.T) {
	g := sampleGraphAIGateOutput()
	if err := ValidateWorkflowGraph(&g); err != nil {
		t.Fatalf("expected valid gate graph: %v", err)
	}
}

func TestExecuteWorkflowDryRun_gateHoldsLowConfidence(t *testing.T) {
	g := sampleGraphAIGateOutput()
	var logs []string
	preview, err := ExecuteWorkflowDryRun(context.Background(), DryRunExecutionInput{
		Graph:       &g,
		Submissions: []string{"short answer"},
		MaxPoints:   10,
		ModelID:     "test/model",
		Runner: lowConfidenceDryRunRunner{},
		Emit: func(ev DryRunEvent) {
			if ev.Type == "log" {
				logs = append(logs, ev.Message)
			}
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if preview.Held == nil || !preview.Held.WouldHold {
		t.Fatalf("expected held preview, got %+v", preview.Held)
	}
	if preview.SuggestedPoints <= 0 {
		t.Fatalf("expected pass-through grade preview, got %.2f", preview.SuggestedPoints)
	}
	joined := strings.Join(logs, "\n")
	if !strings.Contains(joined, "Would hold for review") {
		t.Fatalf("expected would-hold log, got %q", joined)
	}
}

func TestCompileWorkflowGraph_resolvesGradeThroughGate(t *testing.T) {
	g := sampleGraphAIGateOutput()
	compiled, err := CompileWorkflowGraph(&g, "text")
	if err != nil {
		t.Fatal(err)
	}
	if compiled.GradeSource != "ai1" {
		t.Fatalf("grade source = %q, want ai1", compiled.GradeSource)
	}
}

type lowConfidenceDryRunRunner struct{}

func (lowConfidenceDryRunRunner) Score(context.Context, ScoreRequest) (ScoreResult, error) {
	return ScoreResult{Output: GradeOutput{TotalPoints: 8, Confidence: 0.5, Comment: "ok"}}, nil
}

func (lowConfidenceDryRunRunner) RunPrompt(context.Context, string, string, string, string, bool) (string, int, int, float64, error) {
	return `{"total":8,"confidence":0.5,"comment":"ok"}`, 1, 1, 0, nil
}