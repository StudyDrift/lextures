package gradingagent

import (
	"encoding/json"
	"fmt"
	"strings"
)

const (
	WorkflowVersion         = 1
	MaxWorkflowNodes        = 50
	MaxWorkflowEdges        = 100
	NodeTypeOutput          = "output"
	NodeTypeGrader          = "grader"
	NodeTypeAssignmentCtx   = "assignmentContext"
	NodeTypeSubmission      = "submission"
	HandleGrade             = "grade"
	HandleComments          = "comments"
	HandleContext           = "context"
	HandleSubmission        = "submission"
)

// WorkflowGraph is the persisted React Flow graph for the grading agent canvas.
type WorkflowGraph struct {
	Version int            `json:"version"`
	Nodes   []WorkflowNode `json:"nodes"`
	Edges   []WorkflowEdge `json:"edges"`
}

type WorkflowNode struct {
	ID       string         `json:"id"`
	Type     string         `json:"type"`
	Position map[string]any `json:"position"`
	Data     map[string]any `json:"data"`
}

type WorkflowEdge struct {
	ID           string `json:"id"`
	Source       string `json:"source"`
	SourceHandle string `json:"sourceHandle,omitempty"`
	Target       string `json:"target"`
	TargetHandle string `json:"targetHandle,omitempty"`
}

// ValidationError carries a field identifier for client-side highlighting.
type ValidationError struct {
	Field   string
	Message string
}

func (e ValidationError) Error() string { return e.Message }

// CompiledWorkflow holds the result of compiling a graph to scoring inputs.
type CompiledWorkflow struct {
	ScoreRequest ScoreRequest
	GradeSource  string // node id wired to output.grade
	CommentSource string // node id wired to output.comments, may be empty
}

// ParseWorkflowGraph unmarshals and validates raw JSON into a WorkflowGraph.
func ParseWorkflowGraph(raw json.RawMessage) (*WorkflowGraph, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	var g WorkflowGraph
	if err := json.Unmarshal(raw, &g); err != nil {
		return nil, ValidationError{Field: "workflowGraph", Message: "Invalid workflow graph JSON."}
	}
	if err := ValidateWorkflowGraph(&g); err != nil {
		return nil, err
	}
	return &g, nil
}

// ValidateWorkflowGraph checks size caps, node types, edge typing, acyclicity, and required slots.
func ValidateWorkflowGraph(g *WorkflowGraph) error {
	if g == nil {
		return ValidationError{Field: "workflowGraph", Message: "Workflow graph is required."}
	}
	if g.Version != WorkflowVersion {
		return ValidationError{Field: "workflowGraph.version", Message: "Unsupported workflow graph version."}
	}
	if len(g.Nodes) > MaxWorkflowNodes {
		return ValidationError{Field: "workflowGraph.nodes", Message: fmt.Sprintf("Graph exceeds %d node limit.", MaxWorkflowNodes)}
	}
	if len(g.Edges) > MaxWorkflowEdges {
		return ValidationError{Field: "workflowGraph.edges", Message: fmt.Sprintf("Graph exceeds %d edge limit.", MaxWorkflowEdges)}
	}

	outputCount := 0
	nodeByID := make(map[string]WorkflowNode, len(g.Nodes))
	for _, n := range g.Nodes {
		if strings.TrimSpace(n.ID) == "" {
			return ValidationError{Field: "workflowGraph.nodes", Message: "Every node must have an id."}
		}
		if _, dup := nodeByID[n.ID]; dup {
			return ValidationError{Field: "node:" + n.ID, Message: "Duplicate node id."}
		}
		nodeByID[n.ID] = n
		switch n.Type {
		case NodeTypeOutput:
			outputCount++
		case NodeTypeGrader, NodeTypeAssignmentCtx, NodeTypeSubmission:
		default:
			return ValidationError{Field: "node:" + n.ID, Message: "Unknown node type."}
		}
	}
	if outputCount != 1 {
		return ValidationError{Field: "workflowGraph.nodes", Message: "Graph must contain exactly one output node."}
	}

	outputSlotEdges := map[string]string{} // targetHandle -> edge id
	inDegree := make(map[string]int, len(g.Nodes))
	adj := make(map[string][]string, len(g.Nodes))
	for _, e := range g.Edges {
		src, ok := nodeByID[e.Source]
		if !ok {
			return ValidationError{Field: "workflowGraph.edges", Message: "Edge references unknown source node."}
		}
		tgt, ok := nodeByID[e.Target]
		if !ok {
			return ValidationError{Field: "workflowGraph.edges", Message: "Edge references unknown target node."}
		}
		if err := validateEdgeTypes(src, tgt, e); err != nil {
			return err
		}
		if tgt.Type == NodeTypeOutput {
			slot := strings.TrimSpace(e.TargetHandle)
			if slot != HandleGrade && slot != HandleComments {
				return ValidationError{Field: "output", Message: "Output node edges must target grade or comments slots."}
			}
			if _, taken := outputSlotEdges[slot]; taken {
				return ValidationError{Field: "output." + slot, Message: "Each output slot accepts at most one inbound edge."}
			}
			outputSlotEdges[slot] = e.ID
		}
		adj[e.Source] = append(adj[e.Source], e.Target)
		inDegree[e.Target]++
		if _, ok := inDegree[e.Source]; !ok {
			inDegree[e.Source] = inDegree[e.Source]
		}
	}

	if _, ok := outputSlotEdges[HandleGrade]; !ok {
		return ValidationError{Field: "output.grade", Message: "Connect the grade slot before running."}
	}

	if hasCycle(adj, len(g.Nodes)) {
		return ValidationError{Field: "workflowGraph.edges", Message: "Workflow graph must be acyclic."}
	}

	for _, n := range g.Nodes {
		if n.Type != NodeTypeGrader {
			continue
		}
		if strings.TrimSpace(graderPrompt(n)) == "" {
			return ValidationError{Field: "node:" + n.ID + ".prompt", Message: "Grader node prompt is required."}
		}
	}

	return nil
}

func validateEdgeTypes(src, tgt WorkflowNode, e WorkflowEdge) error {
	srcHandle := strings.TrimSpace(e.SourceHandle)
	tgtHandle := strings.TrimSpace(e.TargetHandle)

	switch tgt.Type {
	case NodeTypeOutput:
		if srcHandle != HandleGrade && srcHandle != HandleComments {
			return ValidationError{Field: "output", Message: "Only grade or comments sources may connect to the output node."}
		}
		if tgtHandle != srcHandle {
			return ValidationError{Field: "output." + tgtHandle, Message: "Grade sources must connect to the grade slot; comments sources to the comments slot."}
		}
	case NodeTypeGrader:
		switch tgtHandle {
		case HandleContext:
			if src.Type != NodeTypeAssignmentCtx {
				return ValidationError{Field: "node:" + tgt.ID, Message: "Context input must come from an assignment context node."}
			}
		case HandleSubmission:
			if src.Type != NodeTypeSubmission {
				return ValidationError{Field: "node:" + tgt.ID, Message: "Submission input must come from a submission node."}
			}
		default:
			return ValidationError{Field: "node:" + tgt.ID, Message: "Grader node accepts submission or context inputs only."}
		}
	default:
		return ValidationError{Field: "workflowGraph.edges", Message: "Invalid edge target."}
	}
	return nil
}

func hasCycle(adj map[string][]string, nodeCount int) bool {
	state := make(map[string]int, nodeCount) // 0=unvisited, 1=visiting, 2=done
	var visit func(string) bool
	visit = func(u string) bool {
		if state[u] == 1 {
			return true
		}
		if state[u] == 2 {
			return false
		}
		state[u] = 1
		for _, v := range adj[u] {
			if visit(v) {
				return true
			}
		}
		state[u] = 2
		return false
	}
	for u := range adj {
		if state[u] == 0 && visit(u) {
			return true
		}
	}
	return false
}

// SynthesizeDefaultGraph builds the canonical default graph from legacy prompt/flags.
func SynthesizeDefaultGraph(prompt string, includeContent, includeRubric bool) WorkflowGraph {
	return WorkflowGraph{
		Version: WorkflowVersion,
		Nodes: []WorkflowNode{
			{ID: "output", Type: NodeTypeOutput, Position: map[string]any{"x": 0, "y": 0}, Data: map[string]any{}},
			{ID: "g1", Type: NodeTypeGrader, Position: map[string]any{"x": -320, "y": 0}, Data: map[string]any{
				"prompt": prompt, "modelId": nil,
			}},
			{ID: "ctx", Type: NodeTypeAssignmentCtx, Position: map[string]any{"x": -640, "y": 120}, Data: map[string]any{
				"includeContent": includeContent, "includeRubric": includeRubric,
			}},
			{ID: "sub", Type: NodeTypeSubmission, Position: map[string]any{"x": -640, "y": -80}, Data: map[string]any{}},
		},
		Edges: []WorkflowEdge{
			{ID: "e1", Source: "g1", SourceHandle: HandleGrade, Target: "output", TargetHandle: HandleGrade},
			{ID: "e2", Source: "g1", SourceHandle: HandleComments, Target: "output", TargetHandle: HandleComments},
			{ID: "e3", Source: "ctx", Target: "g1", TargetHandle: HandleContext},
			{ID: "e4", Source: "sub", Target: "g1", TargetHandle: HandleSubmission},
		},
	}
}

// DeriveLegacyFields extracts prompt and include flags from a validated graph.
func DeriveLegacyFields(g *WorkflowGraph) (prompt string, includeContent, includeRubric bool, modelID *string) {
	if g == nil {
		return "", false, false, nil
	}
	nodeByID := make(map[string]WorkflowNode, len(g.Nodes))
	for _, n := range g.Nodes {
		nodeByID[n.ID] = n
	}
	graderID := findGradeGraderNode(g, nodeByID)
	if graderID == "" {
		return "", false, false, nil
	}
	grader := nodeByID[graderID]
	prompt = graderPrompt(grader)
	if mid := graderModelID(grader); mid != "" {
		modelID = &mid
	}
	for _, e := range g.Edges {
		if e.Target != graderID || e.TargetHandle != HandleContext {
			continue
		}
		if ctx, ok := nodeByID[e.Source]; ok && ctx.Type == NodeTypeAssignmentCtx {
			includeContent = boolData(ctx, "includeContent")
			includeRubric = boolData(ctx, "includeRubric")
		}
	}
	return prompt, includeContent, includeRubric, modelID
}

func findGradeGraderNode(g *WorkflowGraph, nodeByID map[string]WorkflowNode) string {
	for _, e := range g.Edges {
		tgt, ok := nodeByID[e.Target]
		if !ok || tgt.Type != NodeTypeOutput || e.TargetHandle != HandleGrade {
			continue
		}
		if src, ok := nodeByID[e.Source]; ok && src.Type == NodeTypeGrader {
			return src.ID
		}
	}
	return ""
}

func graderPrompt(n WorkflowNode) string {
	if n.Data == nil {
		return ""
	}
	if v, ok := n.Data["prompt"].(string); ok {
		return strings.TrimSpace(v)
	}
	return ""
}

func graderModelID(n WorkflowNode) string {
	if n.Data == nil {
		return ""
	}
	switch v := n.Data["modelId"].(type) {
	case string:
		return strings.TrimSpace(v)
	case nil:
		return ""
	default:
		return ""
	}
}

func boolData(n WorkflowNode, key string) bool {
	if n.Data == nil {
		return false
	}
	v, ok := n.Data[key].(bool)
	return ok && v
}

// CompileWorkflowGraph turns a validated graph into a ScoreRequest for the wired grader.
func CompileWorkflowGraph(g *WorkflowGraph, submissionText string) (CompiledWorkflow, error) {
	if err := ValidateWorkflowGraph(g); err != nil {
		return CompiledWorkflow{}, err
	}
	nodeByID := make(map[string]WorkflowNode, len(g.Nodes))
	for _, n := range g.Nodes {
		nodeByID[n.ID] = n
	}
	graderID := findGradeGraderNode(g, nodeByID)
	if graderID == "" {
		return CompiledWorkflow{}, ValidationError{Field: "output.grade", Message: "Connect the grade slot before running."}
	}
	grader := nodeByID[graderID]
	prompt := graderPrompt(grader)
	includeContent := false
	includeRubric := false
	for _, e := range g.Edges {
		if e.Target != graderID || e.TargetHandle != HandleContext {
			continue
		}
		if ctx, ok := nodeByID[e.Source]; ok && ctx.Type == NodeTypeAssignmentCtx {
			includeContent = boolData(ctx, "includeContent")
			includeRubric = boolData(ctx, "includeRubric")
		}
	}
	modelID := graderModelID(grader)
	commentSource := ""
	for _, e := range g.Edges {
		tgt, ok := nodeByID[e.Target]
		if !ok || tgt.Type != NodeTypeOutput || e.TargetHandle != HandleComments {
			continue
		}
		commentSource = e.Source
	}
	req := ScoreRequest{
		InstructorPrompt:         prompt,
		IncludeAssignmentContent: includeContent,
		IncludeRubric:            includeRubric,
		SubmissionText:           submissionText,
	}
	if modelID != "" {
		req.ModelID = modelID
	}
	return CompiledWorkflow{
		ScoreRequest:  req,
		GradeSource:   graderID,
		CommentSource: commentSource,
	}, nil
}

// WorkflowGraphToJSON marshals a graph for API responses and persistence.
func WorkflowGraphToJSON(g *WorkflowGraph) (json.RawMessage, error) {
	if g == nil {
		return nil, nil
	}
	return json.Marshal(g)
}

// EffectiveWorkflowGraph returns stored graph or synthesizes from legacy fields.
func EffectiveWorkflowGraph(stored json.RawMessage, prompt string, includeContent, includeRubric bool) (*WorkflowGraph, error) {
	if len(stored) > 0 {
		return ParseWorkflowGraph(stored)
	}
	g := SynthesizeDefaultGraph(prompt, includeContent, includeRubric)
	return &g, nil
}
