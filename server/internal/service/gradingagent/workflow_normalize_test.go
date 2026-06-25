package gradingagent

import (
	"testing"
)

func legacyAssignmentContextGraph(includeContent, includeRubric bool) WorkflowGraph {
	return WorkflowGraph{
		Version: WorkflowVersion,
		Nodes: []WorkflowNode{
			{ID: "output", Type: NodeTypeOutput, Position: map[string]any{"x": 0, "y": 0}, Data: map[string]any{}},
			{
				ID: "ctx", Type: NodeTypeAssignmentCtx, Position: map[string]any{"x": -640, "y": 0},
				Data: map[string]any{"includeContent": includeContent, "includeRubric": includeRubric},
			},
			{
				ID: "g1", Type: NodeTypeGrader, Position: map[string]any{"x": -320, "y": 0},
				Data: map[string]any{"prompt": "Grade fairly"},
			},
		},
		Edges: []WorkflowEdge{
			{ID: "e-grade", Source: "g1", SourceHandle: HandleGrade, Target: "output", TargetHandle: HandleGrade},
			{ID: "e-context", Source: "ctx", Target: "g1", TargetHandle: HandleContext},
		},
	}
}

func TestNormalizeWorkflowGraph_submissionAlias(t *testing.T) {
	g := WorkflowGraph{
		Version: WorkflowVersion,
		Nodes: []WorkflowNode{
			{ID: "sub", Type: NodeTypeSubmission, Position: map[string]any{"x": 0, "y": 0}, Data: map[string]any{}},
		},
	}
	normalized, changes := NormalizeWorkflowGraph(&g)
	if changes == 0 {
		t.Fatal("expected changes")
	}
	if normalized.Nodes[0].Type != NodeTypeStudentSubmission {
		t.Fatalf("type = %q", normalized.Nodes[0].Type)
	}
	_, againChanges := NormalizeWorkflowGraph(&normalized)
	if againChanges != 0 {
		t.Fatalf("expected idempotent normalize, got %d changes", againChanges)
	}
}

func TestNormalizeWorkflowGraph_assignmentContextExpandsContextHandle(t *testing.T) {
	g := legacyAssignmentContextGraph(true, false)
	normalized, changes := NormalizeWorkflowGraph(&g)
	if changes == 0 {
		t.Fatal("expected changes")
	}
	if normalized.Nodes[1].Type != NodeTypeActivity {
		t.Fatalf("activity type = %q", normalized.Nodes[1].Type)
	}
	var contentEdge *WorkflowEdge
	for i := range normalized.Edges {
		e := normalized.Edges[i]
		if e.Source == "ctx" && e.SourceHandle == HandleContent && e.TargetHandle == HandleContent {
			contentEdge = &normalized.Edges[i]
			break
		}
	}
	if contentEdge == nil {
		t.Fatal("expected content edge from expanded context handle")
	}
	for _, e := range normalized.Edges {
		if e.TargetHandle == HandleContext {
			t.Fatal("context handle should be removed")
		}
	}
}

func TestNormalizeWorkflowGraph_includeFlagsMatchLegacyCompile(t *testing.T) {
	legacy := legacyAssignmentContextGraph(true, true)
	legacyNormalized, _ := NormalizeWorkflowGraph(&legacy)
	legacyCompiled, err := CompileWorkflowGraph(&legacyNormalized, "essay")
	if err != nil {
		t.Fatalf("compile normalized legacy: %v", err)
	}

	canonical := sampleGraphWithGrader("Grade fairly", true, true)
	canonicalCompiled, err := CompileWorkflowGraph(&canonical, "essay")
	if err != nil {
		t.Fatalf("compile canonical: %v", err)
	}

	if legacyCompiled.ScoreRequest.IncludeAssignmentContent != canonicalCompiled.ScoreRequest.IncludeAssignmentContent {
		t.Fatalf("include content mismatch: %v vs %v", legacyCompiled.ScoreRequest.IncludeAssignmentContent, canonicalCompiled.ScoreRequest.IncludeAssignmentContent)
	}
	if legacyCompiled.ScoreRequest.IncludeRubric != canonicalCompiled.ScoreRequest.IncludeRubric {
		t.Fatalf("include rubric mismatch: %v vs %v", legacyCompiled.ScoreRequest.IncludeRubric, canonicalCompiled.ScoreRequest.IncludeRubric)
	}
}

func TestUnmarshalWorkflowGraph_normalizesLegacyTypes(t *testing.T) {
	raw := []byte(`{
		"version": 1,
		"nodes": [
			{"id": "sub", "type": "submission", "position": {"x": 0, "y": 0}, "data": {}}
		],
		"edges": []
	}`)
	g, err := UnmarshalWorkflowGraph(raw)
	if err != nil {
		t.Fatal(err)
	}
	if g.Nodes[0].Type != NodeTypeStudentSubmission {
		t.Fatalf("type = %q", g.Nodes[0].Type)
	}
}
