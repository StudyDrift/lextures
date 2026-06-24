package gradingagent

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/lextures/lextures/server/internal/models/assignmentrubric"
)

func TestAIOutputFormatForNode_rubricWired(t *testing.T) {
	g := sampleGraphWithAI("Grade")
	if AIOutputFormatForNode(&g, "ai1") != AIOutputFormatRubric {
		t.Fatal("expected rubric format when rubric input is wired")
	}
}

func TestAIOutputFormatForNode_scoreOnly(t *testing.T) {
	g := WorkflowGraph{
		Version: WorkflowVersion,
		Nodes: []WorkflowNode{
			{ID: "ai1", Type: NodeTypeAI, Position: map[string]any{"x": 0, "y": 0}, Data: map[string]any{}},
		},
		Edges: []WorkflowEdge{
			{ID: "e1", Source: "sub1", SourceHandle: HandleSubmission, Target: "ai1", TargetHandle: HandleAIInput},
		},
	}
	if AIOutputFormatForNode(&g, "ai1") != AIOutputFormatScore {
		t.Fatal("expected score format without rubric input")
	}
}

func TestBuildAISystemPrompt_includesCriterionIDs(t *testing.T) {
	id := uuid.MustParse("a1b2c3d4-e5f6-7890-abcd-ef1234567890")
	prompt := BuildAISystemPrompt(AIOutputFormatRubric, &assignmentrubric.RubricDefinition{
		Criteria: []assignmentrubric.RubricCriterion{{
			ID:    id,
			Title: "Thesis",
			Levels: []assignmentrubric.RubricLevel{
				{Label: "Weak", Points: 0},
				{Label: "Strong", Points: 4},
			},
		}},
	}, 10)
	if !strings.Contains(prompt, id.String()) {
		t.Fatalf("prompt missing criterion id: %q", prompt)
	}
	if !strings.Contains(prompt, `"total": 8`) {
		t.Fatal("prompt missing score example")
	}
}

func TestBuildCriterionSystemPrompt_includesCriterion(t *testing.T) {
	id := uuid.MustParse("a1b2c3d4-e5f6-7890-abcd-ef1234567890")
	prompt := BuildCriterionSystemPrompt(&assignmentrubric.RubricCriterion{
		ID:    id,
		Title: "Thesis",
		Levels: []assignmentrubric.RubricLevel{
			{Label: "Weak", Points: 0},
			{Label: "Strong", Points: 4},
		},
	})
	if !strings.Contains(prompt, id.String()) {
		t.Fatalf("prompt missing criterion id: %q", prompt)
	}
	if !strings.Contains(prompt, `"score": 4`) {
		t.Fatal("prompt missing score example")
	}
}

func TestParseAIOutput_scoreJSON(t *testing.T) {
	out, err := ParseAIOutput(`{"total":7.5,"comment":"Good","confidence":0.7}`, AIOutputFormatScore, nil, 10)
	if err != nil {
		t.Fatal(err)
	}
	if out.TotalPoints != 7.5 || out.Comment != "Good" {
		t.Fatalf("unexpected output: %+v", out)
	}
}