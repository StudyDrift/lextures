package gradingagent

import (
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/lextures/lextures/server/internal/models/assignmentrubric"
)

func sampleGraphWithRubricAI() WorkflowGraph {
	criterionID := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	return WorkflowGraph{
		Version: WorkflowVersion,
		Nodes: []WorkflowNode{
			{ID: "output", Type: NodeTypeOutput, Position: map[string]any{"x": 0, "y": 0}, Data: map[string]any{}},
			{ID: "ai1", Type: NodeTypeAI, Position: map[string]any{"x": -320, "y": 0}, Data: map[string]any{"prompt": "Grade with rubric"}},
			{ID: "rub1", Type: NodeTypeRubric, Position: map[string]any{"x": -640, "y": 0}, Data: map[string]any{
				"source": "inline",
				"rubric": map[string]any{
					"criteria": []map[string]any{{
						"id": criterionID.String(), "title": "Thesis",
						"levels": []map[string]any{{"label": "Strong", "points": 10.0}},
					}},
				},
			}},
			{ID: "sub1", Type: NodeTypeStudentSubmission, Position: map[string]any{"x": -640, "y": 120}, Data: map[string]any{}},
		},
		Edges: []WorkflowEdge{
			{ID: "e1", Source: "ai1", SourceHandle: HandleAIOutput, Target: "output", TargetHandle: HandleGrade},
			{ID: "e2", Source: "rub1", SourceHandle: HandleRubric, Target: "ai1", TargetHandle: HandleAIInput},
			{ID: "e3", Source: "sub1", SourceHandle: HandleSubmission, Target: "ai1", TargetHandle: HandleAIInput},
		},
	}
}

func TestValidateWorkflowGraph_rejectsRubricToGradeSlot(t *testing.T) {
	g := sampleGraphWithRubricAI()
	g.Edges = append(g.Edges, WorkflowEdge{
		ID: "bad", Source: "rub1", SourceHandle: HandleRubric, Target: "output", TargetHandle: HandleGrade,
	})
	err := ValidateWorkflowGraph(&g)
	if err == nil {
		t.Fatal("expected validation error for rubric → grade")
	}
}

func TestValidateWorkflowGraph_requiresInlineRubricCriteria(t *testing.T) {
	g := sampleGraphWithRubricAI()
	g.Nodes[2].Data = map[string]any{"source": "inline", "rubric": map[string]any{"criteria": []any{}}}
	err := ValidateWorkflowGraph(&g)
	if err == nil {
		t.Fatal("expected validation error for empty inline rubric")
	}
}

func TestExecuteWorkflow_inlineRubricInAIInput(t *testing.T) {
	g := sampleGraphWithRubricAI()
	criterionID := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	var compiledSystemPrompt string
	_, err := ExecuteWorkflow(t.Context(), ExecutionInput{
		Graph:       &g,
		Submissions: []string{"Student essay"},
		MaxPoints:   100,
		ModelID:     "test/model",
		Runner:      stubDryRunRunner{},
		Emit: func(ev ExecutionEvent) {
			if ev.Type == "node_complete" && ev.NodeID == "ai1" {
				compiledSystemPrompt = ev.CompiledSystemPrompt
			}
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(compiledSystemPrompt, criterionID.String()) {
		t.Fatalf("system prompt missing criterion id: %q", compiledSystemPrompt)
	}
}

func TestLoadRubricDefinition_inlineMode(t *testing.T) {
	criterionID := uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb")
	node := WorkflowNode{
		ID: "rub1", Type: NodeTypeRubric,
		Data: map[string]any{
			"source": "inline",
			"rubric": map[string]any{
				"criteria": []map[string]any{{
					"id": criterionID.String(), "title": "Analysis",
					"levels": []map[string]any{{"label": "Good", "points": 5.0}},
				}},
			},
		},
	}
	in := ExecutionInput{}
	rubric, err := in.LoadRubricDefinition(node)
	if err != nil {
		t.Fatal(err)
	}
	if len(rubric.Criteria) != 1 || rubric.Criteria[0].ID != criterionID {
		t.Fatalf("unexpected rubric: %+v", rubric)
	}
}

func TestCompileWorkflowGraph_rubricNodeSetsIncludeRubric(t *testing.T) {
	g := sampleGraphWithRubricAI()
	compiled, err := CompileWorkflowGraph(&g, "essay")
	if err != nil {
		t.Fatal(err)
	}
	if !compiled.ScoreRequest.IncludeRubric {
		t.Fatal("expected includeRubric from wired rubric node")
	}
}

func TestResolveActivity_libraryRubricMode(t *testing.T) {
	libraryID := uuid.MustParse("cccccccc-cccc-cccc-cccc-cccccccccccc")
	criterionID := uuid.MustParse("dddddddd-dddd-dddd-dddd-dddddddddddd")
	node := WorkflowNode{
		ID: "rub1", Type: NodeTypeRubric,
		Data: map[string]any{
			"source":                 "library",
			"rubricAssignmentItemId": libraryID.String(),
		},
	}
	in := ExecutionInput{
		ResolveActivity: func(itemID string) (string, *assignmentrubric.RubricDefinition, error) {
			if itemID != libraryID.String() {
				t.Fatalf("itemID = %q", itemID)
			}
			return "", &assignmentrubric.RubricDefinition{
				Criteria: []assignmentrubric.RubricCriterion{{
					ID: criterionID, Title: "Library criterion",
					Levels: []assignmentrubric.RubricLevel{{Label: "Full", Points: 10}},
				}},
			}, nil
		},
	}
	rubric, err := in.LoadRubricDefinition(node)
	if err != nil {
		t.Fatal(err)
	}
	if rubric.Criteria[0].ID != criterionID {
		t.Fatalf("criterion id mismatch")
	}
}