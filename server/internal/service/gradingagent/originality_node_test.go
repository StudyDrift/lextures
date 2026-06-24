package gradingagent

import (
	"testing"
	"time"
)

func TestResolveOriginalitySignal_similarityFlag(t *testing.T) {
	sim := 55.0
	rows := []OriginalityReportRow{{
		Provider: "turnitin", Status: "done", SimilarityPct: &sim,
		UpdatedAt: time.Date(2026, 6, 20, 12, 0, 0, 0, time.UTC),
	}}
	signal := ResolveOriginalitySignal(OriginalityMetricSimilarity, 0.4, rows)
	if !signal.Present || signal.Score == nil {
		t.Fatal("expected present score")
	}
	if *signal.Score != 0.55 {
		t.Fatalf("score = %v, want 0.55", *signal.Score)
	}
	if !signal.Flag {
		t.Fatal("expected flag true for 0.55 >= 0.4")
	}
}

func TestResolveOriginalitySignal_missingReport(t *testing.T) {
	signal := ResolveOriginalitySignal(OriginalityMetricSimilarity, 0.4, nil)
	if signal.Present {
		t.Fatal("expected absent signal")
	}
	if signal.Flag {
		t.Fatal("expected flag false")
	}
	if signal.Report == "" {
		t.Fatal("expected report message")
	}
}

func TestValidateWorkflowGraph_originalityToFlagSink(t *testing.T) {
	g := WorkflowGraph{
		Version: WorkflowVersion,
		Nodes: []WorkflowNode{
			{ID: "output", Type: NodeTypeOutput},
			{ID: "sub", Type: NodeTypeStudentSubmission},
			{ID: "orig1", Type: NodeTypeOriginality, Data: map[string]any{"metric": "similarity", "flagThreshold": 0.4}},
			{ID: "flag1", Type: NodeTypeFlagForReview, Data: map[string]any{"reasonTemplate": "High similarity"}},
		},
		Edges: []WorkflowEdge{
			{ID: "e1", Source: "sub", SourceHandle: HandleSubmission, Target: "orig1", TargetHandle: HandleSubmission},
			{ID: "e2", Source: "orig1", SourceHandle: HandleFlag, Target: "flag1", TargetHandle: HandleFlag},
		},
	}
	if err := ValidateWorkflowGraph(&g); err != nil {
		t.Fatalf("expected valid originality graph: %v", err)
	}
}