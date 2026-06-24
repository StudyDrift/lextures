package gradingagent

import (
	"context"
	"strings"
	"testing"
)

func sampleGraphRouterFlagElseGrade() WorkflowGraph {
	return WorkflowGraph{
		Version: WorkflowVersion,
		Nodes: []WorkflowNode{
			{ID: "output", Type: NodeTypeOutput},
			{ID: "sub", Type: NodeTypeStudentSubmission},
			{ID: "r1", Type: NodeTypeConditionalRouter, Data: map[string]any{
				"condition": map[string]any{"field": "isEmpty", "operator": "isTrue", "value": true},
			}},
			{ID: "flag1", Type: NodeTypeFlagForReview, Data: map[string]any{
				"queue": "integrity", "priority": "high", "reasonTemplate": "Blank submission",
			}},
			{ID: "ai1", Type: NodeTypeAI, Data: map[string]any{"prompt": "Grade the submission"}},
		},
		Edges: []WorkflowEdge{
			{ID: "e1", Source: "sub", SourceHandle: HandleSubmission, Target: "r1", TargetHandle: HandleAIInput},
			{ID: "e2", Source: "r1", SourceHandle: HandleThen, Target: "flag1", TargetHandle: HandleReason},
			{ID: "e3", Source: "r1", SourceHandle: HandleElse, Target: "ai1", TargetHandle: HandleAIInput},
			{ID: "e4", Source: "ai1", SourceHandle: HandleAIOutput, Target: "output", TargetHandle: HandleGrade},
		},
	}
}

func TestValidateWorkflowGraph_routerFlagAndGradeTerminals(t *testing.T) {
	g := sampleGraphRouterFlagElseGrade()
	if err := ValidateWorkflowGraph(&g); err != nil {
		t.Fatalf("expected valid mixed-terminal graph: %v", err)
	}
}

func TestValidateWorkflowGraph_flagOnlyWithoutGradeSlot(t *testing.T) {
	g := WorkflowGraph{
		Version: WorkflowVersion,
		Nodes: []WorkflowNode{
			{ID: "output", Type: NodeTypeOutput},
			{ID: "sub", Type: NodeTypeStudentSubmission},
			{ID: "flag1", Type: NodeTypeFlagForReview, Data: map[string]any{
				"reasonTemplate": "Needs review",
			}},
		},
		Edges: []WorkflowEdge{
			{ID: "e1", Source: "sub", SourceHandle: HandleSubmission, Target: "flag1", TargetHandle: HandleReason},
		},
	}
	if err := ValidateWorkflowGraph(&g); err != nil {
		t.Fatalf("expected flag-only graph without grade slot to validate: %v", err)
	}
}

func TestExecuteWorkflowDryRun_flagBlankSubmission(t *testing.T) {
	g := sampleGraphRouterFlagElseGrade()
	var logs []string
	preview, err := ExecuteWorkflowDryRun(context.Background(), DryRunExecutionInput{
		Graph:       &g,
		Submissions: []string{""},
		MaxPoints:   10,
		ModelID:     "test/model",
		Runner:      stubDryRunRunner{},
		Emit: func(ev DryRunEvent) {
			if ev.Type == "log" {
				logs = append(logs, ev.Message)
			}
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if preview.Flagged == nil {
		t.Fatal("expected flagged preview")
	}
	if preview.Flagged.Reason != "Blank submission" {
		t.Fatalf("reason = %q", preview.Flagged.Reason)
	}
	if preview.Flagged.Queue != "integrity" || preview.Flagged.Priority != "high" {
		t.Fatalf("queue/priority = %q / %q", preview.Flagged.Queue, preview.Flagged.Priority)
	}
	joined := strings.Join(logs, "\n")
	if !strings.Contains(joined, "Would flag for review") {
		t.Fatalf("expected would-flag log, got %q", joined)
	}
}