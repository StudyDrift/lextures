package httpserver

import (
	"testing"

	gradingagentsvc "github.com/lextures/lextures/server/internal/service/gradingagent"
)

func TestGradingAgentUsesWorkflowEngine(t *testing.T) {
	if !gradingAgentUsesWorkflowEngine(&gradingagentsvc.WorkflowGraph{}) {
		t.Fatal("expected compiled graph to use engine")
	}
	if gradingAgentUsesWorkflowEngine(nil) {
		t.Fatal("expected nil graph to use legacy score")
	}
}

func TestGradingAgentVisionComplexGraphBlocked(t *testing.T) {
	simple := sampleGraphWithGraderForQueueTest("Grade", false, false)
	if gradingAgentVisionComplexGraphBlocked(true, &simple) {
		t.Fatal("simple AI graph should not block vision")
	}

	flagGraph := gradingagentsvc.WorkflowGraph{
		Version: gradingagentsvc.WorkflowVersion,
		Nodes: []gradingagentsvc.WorkflowNode{
			{ID: "output", Type: gradingagentsvc.NodeTypeOutput, Data: map[string]any{}},
			{ID: "flag", Type: gradingagentsvc.NodeTypeFlagForReview, Data: map[string]any{}},
		},
		Edges: []gradingagentsvc.WorkflowEdge{
			{ID: "e1", Source: "flag", SourceHandle: gradingagentsvc.HandleReason, Target: "output", TargetHandle: gradingagentsvc.HandleComments},
		},
	}
	if !gradingAgentVisionComplexGraphBlocked(true, &flagGraph) {
		t.Fatal("vision on flag graph should be blocked")
	}
	if gradingAgentVisionComplexGraphBlocked(false, &flagGraph) {
		t.Fatal("file modality should not block flag graph")
	}
}

func TestGradingAgentPreviewFromDryRun_preservesHeldAndFlagged(t *testing.T) {
	preview := gradingAgentPreviewFromDryRun(gradingagentsvc.DryRunPreview{
		SuggestedPoints: 8,
		Comment:         "Good work",
		Confidence:      0.72,
		Held: &gradingagentsvc.DryRunHeldPreview{
			WouldHold: true,
			Reason:    "Human review gate (below confidence)",
			Queue:     "instructor",
		},
	}, true, "model-x")
	if preview.Held == nil || !preview.Held.WouldHold {
		t.Fatal("expected held preview")
	}
	if !preview.GradedByAI {
		t.Fatal("expected gradedByAI")
	}

	flagged := gradingAgentPreviewFromDryRun(gradingagentsvc.DryRunPreview{
		Flagged: &gradingagentsvc.DryRunFlagPreview{Reason: "Blank submission", Priority: "high"},
	}, false, "")
	if flagged.Flagged == nil || flagged.Flagged.Reason != "Blank submission" {
		t.Fatal("expected flagged preview")
	}
	if flagged.GradedByAI {
		t.Fatal("code-test-only graph should not set gradedByAI")
	}
}

func TestBuildLegacyGradingAgentScoreRequest_substitutesPromptVariables(t *testing.T) {
	g := sampleGraphWithGraderForQueueTest("Grade $Submission.Text", true, false)
	compiled, err := gradingagentsvc.CompileWorkflowGraph(&g, "hello world")
	if err != nil {
		t.Fatal(err)
	}
	d := Deps{}
	req := d.buildLegacyGradingAgentScoreRequest(gradingAgentExecuteInput{
		InstructorPrompt:         "fallback",
		IncludeAssignmentContent: true,
		IncludeRubric:            true,
		SubmissionText:           "hello world",
		Submissions:              []string{"hello world"},
		WorkflowGraph:            &g,
		GradeSourceID:            compiled.GradeSource,
		Compiled:                 compiled,
		ContentMarkdown:          "assignment body",
	}, "model-x")
	if req.ModelID != "model-x" {
		t.Fatalf("model = %q", req.ModelID)
	}
	if req.InstructorPrompt == "fallback" {
		t.Fatal("expected prompt substitution from workflow graph")
	}
}

func sampleGraphWithGraderForQueueTest(prompt string, includeContent, includeRubric bool) gradingagentsvc.WorkflowGraph {
	nodes := []gradingagentsvc.WorkflowNode{
		{ID: "output", Type: gradingagentsvc.NodeTypeOutput, Data: map[string]any{}},
		{ID: "g1", Type: gradingagentsvc.NodeTypeGrader, Data: map[string]any{"prompt": prompt}},
		{ID: "sub1", Type: gradingagentsvc.NodeTypeStudentSubmission, Data: map[string]any{}},
	}
	edges := []gradingagentsvc.WorkflowEdge{
		{ID: "e1", Source: "g1", SourceHandle: gradingagentsvc.HandleGrade, Target: "output", TargetHandle: gradingagentsvc.HandleGrade},
		{ID: "e2", Source: "g1", SourceHandle: gradingagentsvc.HandleComments, Target: "output", TargetHandle: gradingagentsvc.HandleComments},
		{ID: "e3", Source: "sub1", SourceHandle: gradingagentsvc.HandleSubmission, Target: "g1", TargetHandle: gradingagentsvc.HandleSubmission},
	}
	if includeContent || includeRubric {
		nodes = append(nodes, gradingagentsvc.WorkflowNode{
			ID: "act", Type: gradingagentsvc.NodeTypeActivity, Data: map[string]any{},
		})
		if includeContent {
			edges = append(edges, gradingagentsvc.WorkflowEdge{ID: "e4", Source: "act", SourceHandle: gradingagentsvc.HandleContent, Target: "g1", TargetHandle: gradingagentsvc.HandleContent})
		}
		if includeRubric {
			edges = append(edges, gradingagentsvc.WorkflowEdge{ID: "e5", Source: "act", SourceHandle: gradingagentsvc.HandleRubric, Target: "g1", TargetHandle: gradingagentsvc.HandleRubric})
		}
	}
	return gradingagentsvc.WorkflowGraph{Version: gradingagentsvc.WorkflowVersion, Nodes: nodes, Edges: edges}
}
