package gradingagent

import (
	"strings"
	"testing"
)

func TestValidateWorkflowGraph_defaultGraph(t *testing.T) {
	g := SynthesizeDefaultGraph("Grade fairly", true, true)
	g.Nodes[1].Data["prompt"] = "Grade fairly"
	if err := ValidateWorkflowGraph(&g); err != nil {
		t.Fatalf("expected valid default graph: %v", err)
	}
}

func TestValidateWorkflowGraph_rejectsCrossTypeConnection(t *testing.T) {
	g := SynthesizeDefaultGraph("Grade fairly", true, true)
	g.Nodes[1].Data["prompt"] = "Grade fairly"
	g.Edges[1] = WorkflowEdge{ID: "e2", Source: "g1", SourceHandle: HandleGrade, Target: "output", TargetHandle: HandleComments}
	err := ValidateWorkflowGraph(&g)
	if err == nil {
		t.Fatal("expected cross-type rejection")
	}
	ve, ok := err.(ValidationError)
	if !ok || !strings.Contains(ve.Field, "output") {
		t.Fatalf("expected output field error, got %v", err)
	}
}

func TestValidateWorkflowGraph_rejectsUnconnectedGrade(t *testing.T) {
	g := SynthesizeDefaultGraph("Grade fairly", true, true)
	g.Nodes[1].Data["prompt"] = "Grade fairly"
	g.Edges = g.Edges[2:] // remove grade edge
	err := ValidateWorkflowGraph(&g)
	if err == nil {
		t.Fatal("expected unconnected grade slot error")
	}
	ve, ok := err.(ValidationError)
	if !ok || ve.Field != "output.grade" {
		t.Fatalf("expected output.grade, got %v", err)
	}
}

func TestValidateWorkflowGraph_rejectsCycle(t *testing.T) {
	g := SynthesizeDefaultGraph("Grade fairly", true, true)
	g.Nodes[1].Data["prompt"] = "Grade fairly"
	g.Edges = append(g.Edges, WorkflowEdge{ID: "cycle", Source: "output", Target: "g1", TargetHandle: HandleSubmission})
	err := ValidateWorkflowGraph(&g)
	if err == nil {
		t.Fatal("expected cycle rejection")
	}
}

func TestValidateWorkflowGraph_rejectsEmptyPrompt(t *testing.T) {
	g := SynthesizeDefaultGraph("", true, true)
	err := ValidateWorkflowGraph(&g)
	if err == nil {
		t.Fatal("expected empty prompt error")
	}
}

func TestCompileWorkflowGraph(t *testing.T) {
	g := SynthesizeDefaultGraph("Award full marks", true, false)
	g.Nodes[1].Data["prompt"] = "Award full marks"
	compiled, err := CompileWorkflowGraph(&g, "Student essay text.")
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	if compiled.ScoreRequest.InstructorPrompt != "Award full marks" {
		t.Fatalf("prompt mismatch: %q", compiled.ScoreRequest.InstructorPrompt)
	}
	if !compiled.ScoreRequest.IncludeAssignmentContent {
		t.Fatal("expected include content")
	}
	if compiled.ScoreRequest.IncludeRubric {
		t.Fatal("expected includeRubric false")
	}
	if compiled.ScoreRequest.SubmissionText != "Student essay text." {
		t.Fatal("submission text mismatch")
	}
}

func TestDeriveLegacyFields(t *testing.T) {
	g := SynthesizeDefaultGraph("Test prompt", true, true)
	g.Nodes[1].Data["prompt"] = "Test prompt"
	prompt, incContent, incRubric, _ := DeriveLegacyFields(&g)
	if prompt != "Test prompt" || !incContent || !incRubric {
		t.Fatalf("derive mismatch: %q %v %v", prompt, incContent, incRubric)
	}
}

func TestParseWorkflowGraph_legacyFallbackNil(t *testing.T) {
	g, err := ParseWorkflowGraph(nil)
	if err != nil || g != nil {
		t.Fatalf("expected nil graph: %v %v", g, err)
	}
}

func TestEffectiveWorkflowGraph_synthesizes(t *testing.T) {
	g, err := EffectiveWorkflowGraph(nil, "Legacy prompt", false, true)
	if err != nil {
		t.Fatal(err)
	}
	if graderPrompt(g.Nodes[1]) != "Legacy prompt" {
		t.Fatalf("expected legacy prompt in grader node, got %q", graderPrompt(g.Nodes[1]))
	}
}

func TestWorkflowGraph_roundTripJSON(t *testing.T) {
	g := SynthesizeDefaultGraph("Round trip", true, true)
	g.Nodes[1].Data["prompt"] = "Round trip"
	raw, err := WorkflowGraphToJSON(&g)
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := ParseWorkflowGraph(raw)
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Nodes[1].Data["prompt"] != "Round trip" {
		t.Fatal("round trip failed")
	}
}
