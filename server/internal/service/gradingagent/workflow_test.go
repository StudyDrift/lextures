package gradingagent

import (
	"strings"
	"testing"
)

func sampleGraphWithGrader(prompt string, includeContent, includeRubric bool) WorkflowGraph {
	nodes := []WorkflowNode{
		{ID: "output", Type: NodeTypeOutput, Position: map[string]any{"x": 0, "y": 0}, Data: map[string]any{}},
		{ID: "g1", Type: NodeTypeGrader, Position: map[string]any{"x": -320, "y": 0}, Data: map[string]any{
			"prompt": prompt, "modelId": nil,
		}},
	}
	edges := []WorkflowEdge{
		{ID: "e1", Source: "g1", SourceHandle: HandleGrade, Target: "output", TargetHandle: HandleGrade},
		{ID: "e2", Source: "g1", SourceHandle: HandleComments, Target: "output", TargetHandle: HandleComments},
	}
	if includeContent || includeRubric {
		nodes = append(nodes, WorkflowNode{
			ID: "act", Type: NodeTypeActivity, Position: map[string]any{"x": -640, "y": 80}, Data: map[string]any{},
		})
		if includeContent {
			edges = append(edges, WorkflowEdge{ID: "e3", Source: "act", SourceHandle: HandleContent, Target: "g1", TargetHandle: HandleContent})
		}
		if includeRubric {
			edges = append(edges, WorkflowEdge{ID: "e4", Source: "act", SourceHandle: HandleRubric, Target: "g1", TargetHandle: HandleRubric})
		}
	}
	return WorkflowGraph{Version: WorkflowVersion, Nodes: nodes, Edges: edges}
}

func TestSynthesizeDefaultGraph_outputOnly(t *testing.T) {
	g := SynthesizeDefaultGraph("Grade fairly", true, true)
	if len(g.Nodes) != 1 || g.Nodes[0].Type != NodeTypeOutput {
		t.Fatalf("expected output-only graph, got %+v", g.Nodes)
	}
	if len(g.Edges) != 0 {
		t.Fatalf("expected no edges, got %d", len(g.Edges))
	}
	if err := ValidateWorkflowGraph(&g); err == nil {
		t.Fatal("expected validation failure without grade slot")
	}
}

func TestValidateWorkflowGraph_wiredGraderGraph(t *testing.T) {
	g := sampleGraphWithGrader("Grade fairly", true, true)
	if err := ValidateWorkflowGraph(&g); err != nil {
		t.Fatalf("expected valid wired graph: %v", err)
	}
}

func TestValidateWorkflowGraph_rejectsCrossTypeConnection(t *testing.T) {
	g := sampleGraphWithGrader("Grade fairly", true, true)
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
	g := sampleGraphWithGrader("Grade fairly", true, true)
	filtered := make([]WorkflowEdge, 0, len(g.Edges))
	for _, e := range g.Edges {
		if e.TargetHandle == HandleGrade && e.Target == "output" {
			continue
		}
		filtered = append(filtered, e)
	}
	g.Edges = filtered
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
	g := sampleGraphWithGrader("Grade fairly", true, true)
	g.Edges = append(g.Edges, WorkflowEdge{ID: "cycle", Source: "output", Target: "g1", TargetHandle: HandleSubmission})
	err := ValidateWorkflowGraph(&g)
	if err == nil {
		t.Fatal("expected cycle rejection")
	}
}

func TestValidateWorkflowGraph_rejectsEmptyPrompt(t *testing.T) {
	g := sampleGraphWithGrader("", true, true)
	err := ValidateWorkflowGraph(&g)
	if err == nil {
		t.Fatal("expected empty prompt error")
	}
}

func TestCompileWorkflowGraph_activityAssignmentItemIDs(t *testing.T) {
	g := sampleGraphWithGrader("Award full marks", true, true)
	for i := range g.Nodes {
		if g.Nodes[i].ID != "act" {
			continue
		}
		g.Nodes[i].Data = map[string]any{"assignmentItemId": "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"}
	}
	compiled, err := CompileWorkflowGraph(&g, "Student essay text.")
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	if compiled.ContentItemID != "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa" {
		t.Fatalf("content item id = %q", compiled.ContentItemID)
	}
	if compiled.RubricItemID != "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa" {
		t.Fatalf("rubric item id = %q", compiled.RubricItemID)
	}
}

func TestCompileWorkflowGraph(t *testing.T) {
	g := sampleGraphWithGrader("Award full marks", true, false)
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
	g := sampleGraphWithGrader("Test prompt", true, true)
	prompt, incContent, incRubric, _ := DeriveLegacyFields(&g)
	if prompt != "Test prompt" || !incContent || !incRubric {
		t.Fatalf("derive mismatch: %q %v %v", prompt, incContent, incRubric)
	}
}

func sampleGraphWithAI(prompt string) WorkflowGraph {
	return WorkflowGraph{
		Version: WorkflowVersion,
		Nodes: []WorkflowNode{
			{ID: "output", Type: NodeTypeOutput, Position: map[string]any{"x": 0, "y": 0}, Data: map[string]any{}},
			{ID: "ai1", Type: NodeTypeAI, Position: map[string]any{"x": -320, "y": 0}, Data: map[string]any{"prompt": prompt}},
			{ID: "act", Type: NodeTypeActivity, Position: map[string]any{"x": -640, "y": 80}, Data: map[string]any{}},
		},
		Edges: []WorkflowEdge{
			{ID: "e1", Source: "ai1", SourceHandle: HandleAIOutput, Target: "output", TargetHandle: HandleGrade},
			{ID: "e2", Source: "act", SourceHandle: HandleContent, Target: "ai1", TargetHandle: HandleAIInput},
			{ID: "e3", Source: "act", SourceHandle: HandleRubric, Target: "ai1", TargetHandle: HandleAIInput},
		},
	}
}

func TestDeriveLegacyFields_aiWorkflow(t *testing.T) {
	g := sampleGraphWithAI("Grade like a TA")
	prompt, incContent, incRubric, _ := DeriveLegacyFields(&g)
	if prompt != "Grade like a TA" || !incContent || !incRubric {
		t.Fatalf("derive mismatch: %q %v %v", prompt, incContent, incRubric)
	}
}

func TestValidateWorkflowGraphForPersistence_allowsIncompleteDraft(t *testing.T) {
	g := SynthesizeDefaultGraph("", false, false)
	if err := ValidateWorkflowGraphForPersistence(&g); err != nil {
		t.Fatalf("expected draft persistence to allow output-only graph: %v", err)
	}
	if err := ValidateWorkflowGraph(&g); err == nil {
		t.Fatal("expected runnable validation to reject output-only graph")
	}
}

func TestValidateWorkflowGraphForPersistence_allowsQuizResponsesNode(t *testing.T) {
	g := WorkflowGraph{
		Version: WorkflowVersion,
		Nodes: []WorkflowNode{
			{ID: "quizResponses", Type: NodeTypeQuizResponses, Position: map[string]any{"x": -420, "y": 0}, Data: map[string]any{}},
			{ID: "output", Type: NodeTypeOutput, Position: map[string]any{"x": 0, "y": 0}, Data: map[string]any{}},
		},
		Edges: []WorkflowEdge{},
	}
	if err := ValidateWorkflowGraphForPersistence(&g); err != nil {
		t.Fatalf("expected quiz draft graph to persist: %v", err)
	}
}

func TestLoadWorkflowGraph_allowsIncompleteDraft(t *testing.T) {
	g := sampleGraphWithAI("Work in progress")
	raw, err := WorkflowGraphToJSON(&g)
	if err != nil {
		t.Fatal(err)
	}
	loaded, err := LoadWorkflowGraph(raw)
	if err != nil {
		t.Fatalf("load draft graph: %v", err)
	}
	if graderPrompt(loaded.Nodes[1]) != "Work in progress" {
		t.Fatal("prompt not preserved on load")
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
	if len(g.Nodes) != 1 || g.Nodes[0].Type != NodeTypeOutput {
		t.Fatalf("expected output-only synthesized graph, got %+v", g.Nodes)
	}
}

func TestValidateWorkflowGraph_criterionGraderRequiresCriterion(t *testing.T) {
	criterionID := "a1b2c3d4-e5f6-7890-abcd-ef1234567890"
	g := WorkflowGraph{
		Version: WorkflowVersion,
		Nodes: []WorkflowNode{
			{ID: "output", Type: NodeTypeOutput, Position: map[string]any{"x": 0, "y": 0}, Data: map[string]any{}},
			{ID: "cg1", Type: NodeTypeCriterionGrader, Position: map[string]any{"x": -320, "y": 0}, Data: map[string]any{
				"prompt": "Score thesis quality",
			}},
			{ID: "sub1", Type: NodeTypeStudentSubmission, Position: map[string]any{"x": -640, "y": 0}, Data: map[string]any{}},
		},
		Edges: []WorkflowEdge{
			{ID: "e1", Source: "cg1", SourceHandle: HandleGrade, Target: "output", TargetHandle: HandleGrade},
			{ID: "e2", Source: "sub1", SourceHandle: HandleSubmission, Target: "cg1", TargetHandle: HandleSubmission},
		},
	}
	if err := ValidateWorkflowGraph(&g); err == nil {
		t.Fatal("expected missing criterionId validation error")
	}
	g.Nodes[1].Data["criterionId"] = criterionID
	if err := ValidateWorkflowGraph(&g); err != nil {
		t.Fatalf("expected valid graph with criterionId: %v", err)
	}
}

func TestWorkflowGraph_roundTripJSON(t *testing.T) {
	g := sampleGraphWithGrader("Round trip", true, true)
	raw, err := WorkflowGraphToJSON(&g)
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := ParseWorkflowGraph(raw)
	if err != nil {
		t.Fatal(err)
	}
	if graderPrompt(parsed.Nodes[1]) != "Round trip" {
		t.Fatal("round trip failed")
	}
}