package gradingagent

import (
	"encoding/json"
	"fmt"
	"strings"
	"unicode"
)

const (
	WorkflowVersion           = 1
	MaxWorkflowNodes          = 50
	MaxWorkflowEdges          = 100
	NodeTypeOutput            = "output"
	NodeTypeGrader            = "grader"
	NodeTypeAI                = "ai"
	NodeTypeActivity          = "activity"
	NodeTypeStudentSubmission = "studentSubmission"
	NodeTypeAssignmentCtx     = "assignmentContext" // legacy
	NodeTypeSubmission        = "submission"        // legacy
	HandleGrade               = "grade"
	HandleComments            = "comments"
	HandleContent             = "content"
	HandleRubric              = "rubric"
	HandleContext             = "context" // legacy
	HandleSubmission          = "submission"
	HandleAIInput             = "input"
	HandleAIOutput            = "output"
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
	ScoreRequest  ScoreRequest
	GradeSource   string // node id wired to output.grade
	CommentSource string // node id wired to output.comments, may be empty
	ContentItemID string // activity node assignment override for content; empty = grading assignment
	RubricItemID  string // activity node assignment override for rubric; empty = grading assignment
}

func isActivityNodeType(nodeType string) bool {
	return nodeType == NodeTypeActivity || nodeType == NodeTypeAssignmentCtx
}

func isStudentSubmissionNodeType(nodeType string) bool {
	return nodeType == NodeTypeStudentSubmission || nodeType == NodeTypeSubmission
}

func isAINodeType(nodeType string) bool {
	return nodeType == NodeTypeAI
}

func outputSlotSourceIsValid(src WorkflowNode, srcHandle, tgtHandle string) bool {
	switch tgtHandle {
	case HandleGrade:
		if srcHandle == HandleGrade && src.Type == NodeTypeGrader {
			return true
		}
		if srcHandle == HandleAIOutput && src.Type == NodeTypeAI {
			return true
		}
	case HandleComments:
		if srcHandle == HandleComments && src.Type == NodeTypeGrader {
			return true
		}
	}
	return false
}

func aiInputSourceIsValid(sourceType, sourceHandle string) bool {
	if isStudentSubmissionNodeType(sourceType) && sourceHandle == HandleSubmission {
		return true
	}
	if isActivityNodeType(sourceType) && (sourceHandle == HandleContent || sourceHandle == HandleRubric) {
		return true
	}
	if isAINodeType(sourceType) && sourceHandle == HandleAIOutput {
		return true
	}
	return false
}

// UnmarshalWorkflowGraph unmarshals raw JSON without runnable validation.
func UnmarshalWorkflowGraph(raw json.RawMessage) (*WorkflowGraph, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	var g WorkflowGraph
	if err := json.Unmarshal(raw, &g); err != nil {
		return nil, ValidationError{Field: "workflowGraph", Message: "Invalid workflow graph JSON."}
	}
	return &g, nil
}

// ParseWorkflowGraph unmarshals and validates raw JSON into a WorkflowGraph.
func ParseWorkflowGraph(raw json.RawMessage) (*WorkflowGraph, error) {
	g, err := UnmarshalWorkflowGraph(raw)
	if err != nil || g == nil {
		return g, err
	}
	if err := ValidateWorkflowGraph(g); err != nil {
		return nil, err
	}
	return g, nil
}

// LoadWorkflowGraph unmarshals stored JSON and checks persistence constraints only.
func LoadWorkflowGraph(raw json.RawMessage) (*WorkflowGraph, error) {
	g, err := UnmarshalWorkflowGraph(raw)
	if err != nil || g == nil {
		return g, err
	}
	if err := ValidateWorkflowGraphForPersistence(g); err != nil {
		return nil, err
	}
	return g, nil
}

// ValidateWorkflowGraphForPersistence checks draft-save constraints (no runnable requirements).
func ValidateWorkflowGraphForPersistence(g *WorkflowGraph) error {
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
		case NodeTypeOutput, NodeTypeGrader, NodeTypeAI, NodeTypeActivity, NodeTypeStudentSubmission, NodeTypeAssignmentCtx, NodeTypeSubmission:
		default:
			return ValidationError{Field: "node:" + n.ID, Message: "Unknown node type."}
		}
	}

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
		adj[e.Source] = append(adj[e.Source], e.Target)
	}
	if hasCycle(adj, len(g.Nodes)) {
		return ValidationError{Field: "workflowGraph.edges", Message: "Workflow graph must be acyclic."}
	}
	return nil
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
		case NodeTypeGrader, NodeTypeAI, NodeTypeActivity, NodeTypeStudentSubmission, NodeTypeAssignmentCtx, NodeTypeSubmission:
		default:
			return ValidationError{Field: "node:" + n.ID, Message: "Unknown node type."}
		}
	}
	if outputCount != 1 {
		return ValidationError{Field: "workflowGraph.nodes", Message: "Graph must contain exactly one output node."}
	}

	outputSlotEdges := map[string]string{} // targetHandle -> edge id
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
	}

	if _, ok := outputSlotEdges[HandleGrade]; !ok {
		return ValidationError{Field: "output.grade", Message: "Connect the grade slot before running."}
	}

	if hasCycle(adj, len(g.Nodes)) {
		return ValidationError{Field: "workflowGraph.edges", Message: "Workflow graph must be acyclic."}
	}

	for _, n := range g.Nodes {
		if n.Type == NodeTypeGrader && !graderPromptPresent(n) {
			return ValidationError{Field: "node:" + n.ID + ".prompt", Message: "Grader node prompt is required."}
		}
		if n.Type == NodeTypeAI && !graderPromptPresent(n) {
			return ValidationError{Field: "node:" + n.ID + ".prompt", Message: "AI node prompt is required."}
		}
	}

	return nil
}

func validateEdgeTypes(src, tgt WorkflowNode, e WorkflowEdge) error {
	srcHandle := strings.TrimSpace(e.SourceHandle)
	tgtHandle := strings.TrimSpace(e.TargetHandle)

	switch tgt.Type {
	case NodeTypeOutput:
		if tgtHandle != HandleGrade && tgtHandle != HandleComments {
			return ValidationError{Field: "output", Message: "Output node edges must target grade or comments slots."}
		}
		if !outputSlotSourceIsValid(src, srcHandle, tgtHandle) {
			return ValidationError{Field: "output." + tgtHandle, Message: "Grade slot accepts Grader or AI outputs; comments slot accepts Grader comments only."}
		}
	case NodeTypeGrader:
		switch tgtHandle {
		case HandleContent:
			if !isActivityNodeType(src.Type) || srcHandle != HandleContent {
				return ValidationError{Field: "node:" + tgt.ID, Message: "Content input must come from an Activity content output."}
			}
		case HandleRubric:
			if !isActivityNodeType(src.Type) || srcHandle != HandleRubric {
				return ValidationError{Field: "node:" + tgt.ID, Message: "Rubric input must come from an Activity rubric output."}
			}
		case HandleSubmission:
			if !isStudentSubmissionNodeType(src.Type) {
				return ValidationError{Field: "node:" + tgt.ID, Message: "Submission input must come from a Student Submission node."}
			}
		case HandleContext:
			if !isActivityNodeType(src.Type) {
				return ValidationError{Field: "node:" + tgt.ID, Message: "Context input must come from an Activity node."}
			}
		default:
			return ValidationError{Field: "node:" + tgt.ID, Message: "Grader node accepts submission, content, or rubric inputs only."}
		}
	case NodeTypeAI:
		if tgtHandle != HandleAIInput {
			return ValidationError{Field: "node:" + tgt.ID, Message: "AI node edges must target the input slot."}
		}
		if !aiInputSourceIsValid(src.Type, srcHandle) {
			return ValidationError{Field: "node:" + tgt.ID, Message: "AI input must come from a submission, activity, or upstream AI output."}
		}
		if isAINodeType(src.Type) && srcHandle != HandleAIOutput {
			return ValidationError{Field: "node:" + src.ID, Message: "AI node edges must originate from the output slot."}
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

// SynthesizeDefaultGraph builds the canonical empty canvas: fixed output node only.
func SynthesizeDefaultGraph(_ string, _, _ bool) WorkflowGraph {
	return WorkflowGraph{
		Version: WorkflowVersion,
		Nodes: []WorkflowNode{
			{ID: "output", Type: NodeTypeOutput, Position: map[string]any{"x": 0, "y": 0}, Data: map[string]any{}},
		},
		Edges: []WorkflowEdge{},
	}
}

func deriveIncludeFlags(g *WorkflowGraph, promptNodeID string, nodeByID map[string]WorkflowNode) (includeContent, includeRubric bool) {
	promptNode, ok := nodeByID[promptNodeID]
	if !ok {
		return false, false
	}
	if isAINodeType(promptNode.Type) {
		for _, e := range g.Edges {
			if e.Target != promptNodeID || strings.TrimSpace(e.TargetHandle) != HandleAIInput {
				continue
			}
			src, ok := nodeByID[e.Source]
			if !ok || !isActivityNodeType(src.Type) {
				continue
			}
			switch strings.TrimSpace(e.SourceHandle) {
			case HandleContent:
				includeContent = true
			case HandleRubric:
				includeRubric = true
			}
		}
		return includeContent, includeRubric
	}
	for _, e := range g.Edges {
		if e.Target != promptNodeID {
			continue
		}
		switch e.TargetHandle {
		case HandleContent:
			if src, ok := nodeByID[e.Source]; ok && isActivityNodeType(src.Type) {
				includeContent = true
			}
		case HandleRubric:
			if src, ok := nodeByID[e.Source]; ok && isActivityNodeType(src.Type) {
				includeRubric = true
			}
		case HandleContext:
			if ctx, ok := nodeByID[e.Source]; ok && isActivityNodeType(ctx.Type) {
				if ctx.Type == NodeTypeAssignmentCtx {
					includeContent = boolData(ctx, "includeContent")
					includeRubric = boolData(ctx, "includeRubric")
				} else {
					includeContent = true
					includeRubric = true
				}
			}
		}
	}
	return includeContent, includeRubric
}

// DeriveLegacyFields extracts prompt and include flags from a workflow graph.
func DeriveLegacyFields(g *WorkflowGraph) (prompt string, includeContent, includeRubric bool, modelID *string) {
	if g == nil {
		return "", false, false, nil
	}
	nodeByID := make(map[string]WorkflowNode, len(g.Nodes))
	for _, n := range g.Nodes {
		nodeByID[n.ID] = n
	}
	promptNodeID := findWorkflowPromptNode(g, nodeByID)
	if promptNodeID == "" {
		return "", false, false, nil
	}
	promptNode := nodeByID[promptNodeID]
	prompt = graderPrompt(promptNode)
	if mid := graderModelID(promptNode); mid != "" {
		modelID = &mid
	}
	includeContent, includeRubric = deriveIncludeFlags(g, promptNodeID, nodeByID)
	return prompt, includeContent, includeRubric, modelID
}

// PersistencePrompt returns a non-empty legacy prompt for config storage.
func PersistencePrompt(g *WorkflowGraph, fallback string) string {
	if derived, _, _, _ := DeriveLegacyFields(g); strings.TrimSpace(derived) != "" {
		return derived
	}
	if trimmed := strings.TrimSpace(fallback); trimmed != "" {
		return trimmed
	}
	return "."
}

func findWorkflowPromptNode(g *WorkflowGraph, nodeByID map[string]WorkflowNode) string {
	for _, e := range g.Edges {
		tgt, ok := nodeByID[e.Target]
		if !ok || tgt.Type != NodeTypeOutput || e.TargetHandle != HandleGrade {
			continue
		}
		if src, ok := nodeByID[e.Source]; ok && (src.Type == NodeTypeGrader || src.Type == NodeTypeAI) {
			return src.ID
		}
	}
	for _, n := range g.Nodes {
		if n.Type == NodeTypeGrader || n.Type == NodeTypeAI {
			return n.ID
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

func graderPromptPresent(n WorkflowNode) bool {
	prompt := graderPrompt(n)
	if prompt == "" {
		return false
	}
	for _, r := range prompt {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			return true
		}
	}
	return false
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

func activityAssignmentItemID(n WorkflowNode) string {
	if n.Data == nil {
		return ""
	}
	if v, ok := n.Data["assignmentItemId"].(string); ok {
		return strings.TrimSpace(v)
	}
	return ""
}

func resolveWiredActivityItemIDs(g *WorkflowGraph, promptNodeID string, nodeByID map[string]WorkflowNode) (contentItemID, rubricItemID string) {
	promptNode, ok := nodeByID[promptNodeID]
	if !ok {
		return "", ""
	}
	for _, e := range g.Edges {
		if e.Target != promptNodeID {
			continue
		}
		src, ok := nodeByID[e.Source]
		if !ok || !isActivityNodeType(src.Type) {
			continue
		}
		targetHandle := strings.TrimSpace(e.TargetHandle)
		sourceHandle := strings.TrimSpace(e.SourceHandle)
		switch targetHandle {
		case HandleContent:
			contentItemID = activityAssignmentItemID(src)
		case HandleRubric:
			rubricItemID = activityAssignmentItemID(src)
		case HandleContext:
			if src.Type == NodeTypeAssignmentCtx {
				continue
			}
			if contentItemID == "" {
				contentItemID = activityAssignmentItemID(src)
			}
			if rubricItemID == "" {
				rubricItemID = activityAssignmentItemID(src)
			}
		case HandleAIInput:
			if !isAINodeType(promptNode.Type) {
				continue
			}
			switch sourceHandle {
			case HandleContent:
				contentItemID = activityAssignmentItemID(src)
			case HandleRubric:
				rubricItemID = activityAssignmentItemID(src)
			}
		}
	}
	return contentItemID, rubricItemID
}

func findGradeSourceNode(g *WorkflowGraph, nodeByID map[string]WorkflowNode) string {
	for _, e := range g.Edges {
		tgt, ok := nodeByID[e.Target]
		if !ok || tgt.Type != NodeTypeOutput || e.TargetHandle != HandleGrade {
			continue
		}
		if src, ok := nodeByID[e.Source]; ok && (src.Type == NodeTypeGrader || src.Type == NodeTypeAI) {
			return src.ID
		}
	}
	return ""
}

// CompileWorkflowGraph turns a validated graph into a ScoreRequest for the wired grade source.
func CompileWorkflowGraph(g *WorkflowGraph, submissionText string) (CompiledWorkflow, error) {
	if err := ValidateWorkflowGraph(g); err != nil {
		return CompiledWorkflow{}, err
	}
	nodeByID := make(map[string]WorkflowNode, len(g.Nodes))
	for _, n := range g.Nodes {
		nodeByID[n.ID] = n
	}
	gradeSourceID := findGradeSourceNode(g, nodeByID)
	if gradeSourceID == "" {
		return CompiledWorkflow{}, ValidationError{Field: "output.grade", Message: "Connect the grade slot before running."}
	}
	gradeSource := nodeByID[gradeSourceID]
	prompt := graderPrompt(gradeSource)
	includeContent, includeRubric := deriveIncludeFlags(g, gradeSourceID, nodeByID)
	modelID := graderModelID(gradeSource)
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
	contentItemID, rubricItemID := resolveWiredActivityItemIDs(g, gradeSourceID, nodeByID)
	return CompiledWorkflow{
		ScoreRequest:  req,
		GradeSource:   gradeSourceID,
		CommentSource: commentSource,
		ContentItemID: contentItemID,
		RubricItemID:  rubricItemID,
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
		return LoadWorkflowGraph(stored)
	}
	g := SynthesizeDefaultGraph(prompt, includeContent, includeRubric)
	return &g, nil
}
