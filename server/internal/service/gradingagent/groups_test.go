package gradingagent

import (
	"context"
	"testing"
)

// groupedTieredGraph wraps the canonical "Q1 tiered scoring" routers+setScore
// cluster into a group with one input (the quiz question) and one Grade output.
func groupedTieredGraph() WorkflowGraph {
	subgraph := map[string]any{
		"version": 1,
		"nodes": []map[string]any{
			{"id": "rtr1", "type": "conditionalRouter", "position": map[string]any{"x": 0, "y": 0},
				"data": map[string]any{"condition": map[string]any{"field": "submissionText", "operator": "contains", "value": "x"}}},
			{"id": "ss1", "type": "setScore", "position": map[string]any{"x": 200, "y": 0}, "data": map[string]any{"score": 12}},
			{"id": "ss0", "type": "setScore", "position": map[string]any{"x": 200, "y": 120}, "data": map[string]any{"score": 0}},
		},
		"edges": []map[string]any{
			{"id": "ie1", "source": "rtr1", "sourceHandle": "then", "target": "ss1", "targetHandle": "grade"},
			{"id": "ie2", "source": "rtr1", "sourceHandle": "else", "target": "ss0", "targetHandle": "grade"},
		},
	}
	return WorkflowGraph{
		Version: WorkflowVersion,
		Nodes: []WorkflowNode{
			{ID: "sub", Type: NodeTypeStudentSubmission, Position: map[string]any{"x": -600, "y": 0}, Data: map[string]any{}},
			{ID: "grp1", Type: NodeTypeGroup, Position: map[string]any{"x": -200, "y": 0}, Data: map[string]any{
				"label":    "Tiered scoring",
				"subgraph": subgraph,
				"inputs":   []map[string]any{{"id": "in1", "label": "Submission", "nodeId": "rtr1", "handle": "input"}},
				"outputs": []map[string]any{
					{"id": "outThen", "label": "Full", "nodeId": "ss1", "handle": "grade"},
					{"id": "outElse", "label": "None", "nodeId": "ss0", "handle": "grade"},
				},
			}},
			{ID: "output", Type: NodeTypeOutput, Position: map[string]any{"x": 200, "y": 0}, Data: map[string]any{}},
		},
		// Both branches feed the assignment grade slot; only one is active at runtime.
		Edges: []WorkflowEdge{
			{ID: "e1", Source: "sub", SourceHandle: "submission", Target: "grp1", TargetHandle: "in1"},
			{ID: "e2", Source: "grp1", SourceHandle: "outThen", Target: "output", TargetHandle: "grade"},
			{ID: "e3", Source: "grp1", SourceHandle: "outElse", Target: "output", TargetHandle: "grade"},
		},
	}
}

func TestFlattenWorkflowGraph_inlinesGroup(t *testing.T) {
	g := groupedTieredGraph()
	flat, err := FlattenWorkflowGraph(&g)
	if err != nil {
		t.Fatalf("flatten error: %v", err)
	}
	for _, n := range flat.Nodes {
		if n.Type == NodeTypeGroup {
			t.Fatal("flattened graph should not contain group nodes")
		}
	}
	// Internal nodes are prefixed.
	ids := map[string]bool{}
	for _, n := range flat.Nodes {
		ids[n.ID] = true
	}
	if !ids["grp1/rtr1"] || !ids["grp1/ss1"] {
		t.Fatalf("expected prefixed internal nodes, got %v", ids)
	}
	// Boundary input edge rewired to the internal router input.
	var foundIn, foundOut bool
	for _, e := range flat.Edges {
		if e.Source == "sub" && e.SourceHandle == "submission" && e.Target == "grp1/rtr1" && e.TargetHandle == "input" {
			foundIn = true
		}
		if e.Source == "grp1/ss1" && e.SourceHandle == "grade" && e.Target == "output" && e.TargetHandle == "grade" {
			foundOut = true
		}
	}
	if !foundIn {
		t.Fatal("input boundary edge was not rewired to the internal node")
	}
	if !foundOut {
		t.Fatal("output boundary edge was not rewired from the internal node")
	}
}

func TestValidateWorkflowGraph_groupedGraphIsRunnable(t *testing.T) {
	g := groupedTieredGraph()
	if err := ValidateWorkflowGraph(&g); err != nil {
		t.Fatalf("expected grouped graph to be runnable: %v", err)
	}
}

func TestValidateWorkflowGraphForPersistence_acceptsGroup(t *testing.T) {
	g := groupedTieredGraph()
	if err := ValidateWorkflowGraphForPersistence(&g); err != nil {
		t.Fatalf("persistence validation should accept groups: %v", err)
	}
}

func TestValidateGroupStructure_rejectsDanglingPort(t *testing.T) {
	g := groupedTieredGraph()
	for i := range g.Nodes {
		if g.Nodes[i].ID == "grp1" {
			data := g.Nodes[i].Data
			data["inputs"] = []map[string]any{{"id": "in1", "nodeId": "doesNotExist", "handle": "input"}}
		}
	}
	if err := ValidateWorkflowGraphForPersistence(&g); err == nil {
		t.Fatal("expected error for port referencing a missing internal node")
	}
}

func TestExecuteWorkflow_groupedTieredScoring(t *testing.T) {
	g := groupedTieredGraph()
	// "then" branch active (submission contains "x") routes to Set Score 12.
	preview, err := ExecuteWorkflow(context.Background(), ExecutionInput{
		Graph:       &g,
		Submissions: []string{"xenon"},
		MaxPoints:   12,
	})
	if err != nil {
		t.Fatalf("execute error: %v", err)
	}
	if preview.SuggestedPoints != 12 {
		t.Fatalf("expected 12 points from the then branch, got %v", preview.SuggestedPoints)
	}
}

func TestWorkflowUsesLLM_detectsInsideGroup(t *testing.T) {
	sub := map[string]any{
		"version": 1,
		"nodes": []map[string]any{
			{"id": "ai1", "type": "ai", "position": map[string]any{"x": 0, "y": 0}, "data": map[string]any{"prompt": "grade"}},
		},
		"edges": []map[string]any{},
	}
	g := WorkflowGraph{
		Version: WorkflowVersion,
		Nodes: []WorkflowNode{
			{ID: "grp1", Type: NodeTypeGroup, Position: map[string]any{"x": 0, "y": 0}, Data: map[string]any{
				"subgraph": sub,
				"outputs":  []map[string]any{{"id": "o1", "nodeId": "ai1", "handle": "output"}},
			}},
		},
	}
	if !WorkflowUsesLLM(&g) {
		t.Fatal("expected LLM usage to be detected inside the group")
	}
}
