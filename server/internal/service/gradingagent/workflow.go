package gradingagent

import (
	"encoding/json"
	"fmt"
	"strings"
	"unicode"

	"github.com/google/uuid"
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
	NodeTypeQuizResponses     = "quizResponses"
	NodeTypeCodeTestRunner    = "codeTestRunner"
	NodeTypeConditionalRouter = "conditionalRouter"
	NodeTypeCriterionGrader   = "criterionGrader"
	NodeTypeFlagForReview     = "flagForReview"
	NodeTypeHumanReviewGate   = "humanReviewGate"
	NodeTypeOriginality       = "originality"
	NodeTypeReference         = "reference"
	NodeTypeRubric            = "rubric"
	NodeTypeScoreAggregator   = "scoreAggregator"
	NodeTypeSetScore          = "setScore"
	NodeTypeAssignmentCtx     = "assignmentContext" // legacy
	NodeTypeSubmission        = "submission"        // legacy
	HandleGrade               = "grade"
	HandleReport              = "report"
	HandleScore               = "score"
	HandleComments            = "comments"
	HandleContent             = "content"
	HandleRubric              = "rubric"
	HandleContext             = "context" // legacy
	HandleSubmission          = "submission"
	HandleAIInput             = "input"
	HandleAIOutput            = "output"
	HandleThen                = "then"
	HandleElse                = "else"
	HandleReason              = "reason"
	HandleFlag                = "flag"
	HandleReference           = "reference"
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
	return nodeType == NodeTypeActivity
}

func isStudentSubmissionNodeType(nodeType string) bool {
	return nodeType == NodeTypeStudentSubmission
}

func isQuizResponsesNodeType(nodeType string) bool {
	return nodeType == NodeTypeQuizResponses
}

func isAINodeType(nodeType string) bool {
	return nodeType == NodeTypeAI
}

func isCodeTestRunnerNodeType(nodeType string) bool {
	return nodeType == NodeTypeCodeTestRunner
}

func isCriterionGraderNodeType(nodeType string) bool {
	return nodeType == NodeTypeCriterionGrader
}

func outputSlotSourceIsValid(src WorkflowNode, srcHandle, tgtHandle string) bool {
	if isQuizGradeHandle(tgtHandle) {
		if srcHandle == HandleGrade && (src.Type == NodeTypeGrader || isCriterionGraderNodeType(src.Type)) {
			return true
		}
		if srcHandle == HandleAIOutput && src.Type == NodeTypeAI {
			return true
		}
		if srcHandle == HandleGrade && isCodeTestRunnerNodeType(src.Type) {
			return true
		}
		if (srcHandle == HandleThen || srcHandle == HandleElse) && isConditionalRouterNodeType(src.Type) {
			return true
		}
		if srcHandle == HandleGrade && isHumanReviewGateNodeType(src.Type) {
			return true
		}
		if srcHandle == HandleGrade && isScoreAggregatorNodeType(src.Type) {
			return true
		}
		if srcHandle == HandleGrade && isSetScoreNodeType(src.Type) {
			return true
		}
		return false
	}
	switch tgtHandle {
	case HandleGrade:
		if srcHandle == HandleGrade && (src.Type == NodeTypeGrader || isCriterionGraderNodeType(src.Type)) {
			return true
		}
		if srcHandle == HandleAIOutput && src.Type == NodeTypeAI {
			return true
		}
		if srcHandle == HandleGrade && isCodeTestRunnerNodeType(src.Type) {
			return true
		}
		if (srcHandle == HandleThen || srcHandle == HandleElse) && isConditionalRouterNodeType(src.Type) {
			return true
		}
		if srcHandle == HandleGrade && isHumanReviewGateNodeType(src.Type) {
			return true
		}
		if srcHandle == HandleGrade && isScoreAggregatorNodeType(src.Type) {
			return true
		}
		if srcHandle == HandleGrade && isSetScoreNodeType(src.Type) {
			return true
		}
	case HandleComments:
		if srcHandle == HandleComments && isScoreAggregatorNodeType(src.Type) {
			return true
		}
		if srcHandle == HandleComments && (src.Type == NodeTypeGrader || isCriterionGraderNodeType(src.Type)) {
			return true
		}
		if srcHandle == HandleReport && isCodeTestRunnerNodeType(src.Type) {
			return true
		}
	}
	return false
}

func aiInputSourceIsValid(sourceType, sourceHandle string) bool {
	if isStudentSubmissionNodeType(sourceType) && sourceHandle == HandleSubmission {
		return true
	}
	if isQuizResponsesNodeType(sourceType) && isQuizQuestionHandle(sourceHandle) {
		return true
	}
	if isActivityNodeType(sourceType) && (sourceHandle == HandleContent || sourceHandle == HandleRubric) {
		return true
	}
	if isAINodeType(sourceType) && sourceHandle == HandleAIOutput {
		return true
	}
	if isConditionalRouterNodeType(sourceType) && (sourceHandle == HandleThen || sourceHandle == HandleElse) {
		return true
	}
	if isOriginalityNodeType(sourceType) && (sourceHandle == HandleScore || sourceHandle == HandleReport) {
		return true
	}
	if isReferenceNodeType(sourceType) && sourceHandle == HandleReference {
		return true
	}
	if isRubricNodeType(sourceType) && sourceHandle == HandleRubric {
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
	normalized, _ := NormalizeWorkflowGraph(&g)
	return &normalized, nil
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
		case NodeTypeOutput, NodeTypeGrader, NodeTypeCriterionGrader, NodeTypeAI, NodeTypeActivity, NodeTypeStudentSubmission, NodeTypeQuizResponses, NodeTypeCodeTestRunner, NodeTypeConditionalRouter, NodeTypeFlagForReview, NodeTypeHumanReviewGate, NodeTypeOriginality, NodeTypeReference, NodeTypeRubric, NodeTypeScoreAggregator, NodeTypeSetScore:
		case NodeTypeGroup:
			gd, err := parseGroupData(n)
			if err != nil {
				return err
			}
			if err := validateGroupStructure(n.ID, gd); err != nil {
				return err
			}
			sub := gd.Subgraph
			if err := ValidateWorkflowGraphForPersistence(&sub); err != nil {
				return err
			}
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
		// Edges at a group boundary are typed by the group's ports, not node handles;
		// they are validated structurally when the group is flattened.
		if !isGroupNodeType(src.Type) && !isGroupNodeType(tgt.Type) {
			if err := validateEdgeTypes(src, tgt, e); err != nil {
				return err
			}
		}
		adj[e.Source] = append(adj[e.Source], e.Target)
	}
	if hasCycle(adj, len(g.Nodes)) {
		return ValidationError{Field: "workflowGraph.edges", Message: "Workflow graph must be acyclic."}
	}
	return nil
}

// ValidateWorkflowGraph checks size caps, node types, edge typing, acyclicity, and required slots.
// Group nodes are flattened first so runnable validation runs on the fully-expanded graph.
func ValidateWorkflowGraph(g *WorkflowGraph) error {
	if g == nil {
		return ValidationError{Field: "workflowGraph", Message: "Workflow graph is required."}
	}
	if graphContainsGroup(g) {
		flat, err := FlattenWorkflowGraph(g)
		if err != nil {
			return err
		}
		return ValidateWorkflowGraph(&flat)
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
		case NodeTypeGrader, NodeTypeCriterionGrader, NodeTypeAI, NodeTypeActivity, NodeTypeStudentSubmission, NodeTypeQuizResponses, NodeTypeCodeTestRunner, NodeTypeConditionalRouter, NodeTypeFlagForReview, NodeTypeHumanReviewGate, NodeTypeOriginality, NodeTypeReference, NodeTypeRubric, NodeTypeScoreAggregator, NodeTypeSetScore:
		default:
			return ValidationError{Field: "node:" + n.ID, Message: "Unknown node type."}
		}
	}
	if outputCount != 1 {
		return ValidationError{Field: "workflowGraph.nodes", Message: "Graph must contain exactly one output node."}
	}

	outputSlotEdges := map[string][]string{} // targetHandle -> edge ids
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
			if slot != HandleGrade && slot != HandleComments && !isQuizGradeHandle(slot) {
				return ValidationError{Field: "output", Message: "Output node edges must target grade or comments slots."}
			}
			if len(outputSlotEdges[slot]) > 0 && (slot == HandleComments || isQuizGradeHandle(slot)) {
				return ValidationError{Field: "output." + slot, Message: "Each output slot accepts at most one inbound edge."}
			}
			outputSlotEdges[slot] = append(outputSlotEdges[slot], e.ID)
		}
		adj[e.Source] = append(adj[e.Source], e.Target)
	}

	if graphIsQuizMode(g) {
		hasQuizGrade := false
		for slot, edges := range outputSlotEdges {
			if isQuizGradeHandle(slot) && len(edges) > 0 {
				hasQuizGrade = true
				break
			}
		}
		if !hasQuizGrade && !graphHasFlagSink(g) {
			return ValidationError{Field: "output.grade", Message: "Connect at least one question grade slot before running."}
		}
	} else if len(outputSlotEdges[HandleGrade]) == 0 && !graphHasFlagSink(g) {
		return ValidationError{Field: "output.grade", Message: "Connect the grade slot before running."}
	}

	if hasCycle(adj, len(g.Nodes)) {
		return ValidationError{Field: "workflowGraph.edges", Message: "Workflow graph must be acyclic."}
	}

	for _, n := range g.Nodes {
		if n.Type == NodeTypeGrader && !graderPromptPresent(n) {
			return ValidationError{Field: "node:" + n.ID + ".prompt", Message: "Grader node prompt is required."}
		}
		if n.Type == NodeTypeCriterionGrader {
			if !graderPromptPresent(n) {
				return ValidationError{Field: "node:" + n.ID + ".prompt", Message: "Criterion Grader prompt is required."}
			}
			if _, err := criterionIDFromNode(n); err != nil {
				return err
			}
		}
		if n.Type == NodeTypeAI && !graderPromptPresent(n) {
			return ValidationError{Field: "node:" + n.ID + ".prompt", Message: "AI node prompt is required."}
		}
		if n.Type == NodeTypeCodeTestRunner && !codeTestRunnerHasConfig(n) {
			return ValidationError{Field: "node:" + n.ID + ".testCases", Message: "Add at least one test case or select a test suite."}
		}
		if n.Type == NodeTypeConditionalRouter {
			if _, err := routerConditionFromNode(n); err != nil {
				return ValidationError{Field: "node:" + n.ID + ".condition", Message: err.Error()}
			}
		}
		if n.Type == NodeTypeHumanReviewGate && !gateHasGradeInput(g, n.ID) {
			return ValidationError{Field: "node:" + n.ID + ".grade", Message: "Connect a grade input to the Human Review Gate."}
		}
		if n.Type == NodeTypeOriginality && !originalityHasSubmissionInput(g, n.ID) {
			return ValidationError{Field: "node:" + n.ID + ".submission", Message: "Connect a submission input to the Originality Check."}
		}
		if n.Type == NodeTypeReference && !referenceHasSource(n) {
			return ValidationError{Field: "node:" + n.ID + ".text", Message: "Add reference text or select a course file."}
		}
		if n.Type == NodeTypeRubric && !rubricHasSource(n) {
			return ValidationError{Field: "node:" + n.ID + ".source", Message: "Configure a rubric source for this node."}
		}
		if n.Type == NodeTypeScoreAggregator {
			if !aggregatorHasGradeInput(g, n.ID) {
				return ValidationError{Field: "node:" + n.ID + ".grade", Message: "Connect at least one grade input to the Score Aggregator."}
			}
			if aggregatorModeFromNode(n) == AggregatorModeRubricMerge {
				if dupes := DetectRubricMergeCriterionConflicts(wiredAggregatorSourceCriterionIDs(g, n.ID, nodeByID)); len(dupes) > 0 {
					return ValidationError{Field: "node:" + n.ID + ".mode", Message: "rubricMerge: each criterion may be scored only once across inputs."}
				}
			}
		}
	}

	if err := validateRouterFieldAvailability(g, nodeByID); err != nil {
		return err
	}
	if err := validateRouterPathReachability(g, nodeByID); err != nil {
		return err
	}

	return nil
}

func validateEdgeTypes(src, tgt WorkflowNode, e WorkflowEdge) error {
	srcHandle := strings.TrimSpace(e.SourceHandle)
	tgtHandle := strings.TrimSpace(e.TargetHandle)

	switch tgt.Type {
	case NodeTypeOutput:
		if tgtHandle != HandleGrade && tgtHandle != HandleComments && !isQuizGradeHandle(tgtHandle) {
			return ValidationError{Field: "output", Message: "Output node edges must target grade or comments slots."}
		}
		if !outputSlotSourceIsValid(src, srcHandle, tgtHandle) {
			return ValidationError{Field: "output." + tgtHandle, Message: "Grade slot accepts Grader, Criterion Grader, AI, Code Test Runner, or Conditional Router branch outputs; comments slot accepts Grader or Criterion Grader comments or test reports."}
		}
	case NodeTypeGrader, NodeTypeCriterionGrader:
		switch tgtHandle {
		case HandleContent:
			if (isActivityNodeType(src.Type) && srcHandle == HandleContent) || referenceContentSourceIsValid(src, srcHandle) {
				break
			}
			return ValidationError{Field: "node:" + tgt.ID, Message: "Content input must come from an Activity content output or Reference Material."}
		case HandleRubric:
			if (isActivityNodeType(src.Type) && srcHandle == HandleRubric) || rubricOutputSourceIsValid(src, srcHandle) {
				break
			}
			return ValidationError{Field: "node:" + tgt.ID, Message: "Rubric input must come from an Activity or Rubric rubric output."}
		case HandleSubmission:
			if !quizSubmissionSourceValid(src, srcHandle) {
				return ValidationError{Field: "node:" + tgt.ID, Message: "Submission input must come from a Student Submission or Quiz Responses node."}
			}
		default:
			msg := "Grader node accepts submission, content, or rubric inputs only."
			if isCriterionGraderNodeType(tgt.Type) {
				msg = "Criterion Grader accepts submission, content, or rubric inputs only."
			}
			return ValidationError{Field: "node:" + tgt.ID, Message: msg}
		}
	case NodeTypeAI:
		if tgtHandle != HandleAIInput {
			return ValidationError{Field: "node:" + tgt.ID, Message: "AI node edges must target the input slot."}
		}
		if !aiInputSourceIsValid(src.Type, srcHandle) {
			return ValidationError{Field: "node:" + tgt.ID, Message: "AI input must come from a submission, activity, reference, or upstream AI output."}
		}
		if isAINodeType(src.Type) && srcHandle != HandleAIOutput {
			return ValidationError{Field: "node:" + src.ID, Message: "AI node edges must originate from the output slot."}
		}
	case NodeTypeCodeTestRunner:
		if tgtHandle != HandleSubmission {
			return ValidationError{Field: "node:" + tgt.ID, Message: "Code Test Runner accepts a submission input only."}
		}
		if !quizSubmissionSourceValid(src, srcHandle) {
			return ValidationError{Field: "node:" + tgt.ID, Message: "Submission input must come from a Student Submission or Quiz Responses node."}
		}
	case NodeTypeConditionalRouter:
		if tgtHandle != HandleAIInput {
			return ValidationError{Field: "node:" + tgt.ID, Message: "Conditional Router edges must target the input slot."}
		}
		if !routerInputSourceIsValid(src.Type, srcHandle) {
			return ValidationError{Field: "node:" + tgt.ID, Message: "Router input must come from a submission, grade, or upstream branch output."}
		}
		if isConditionalRouterNodeType(src.Type) && srcHandle != HandleThen && srcHandle != HandleElse {
			return ValidationError{Field: "node:" + src.ID, Message: "Conditional Router edges must originate from then or else outputs."}
		}
	case NodeTypeFlagForReview:
		if tgtHandle != HandleReason && tgtHandle != HandleComments && tgtHandle != HandleReport &&
			tgtHandle != HandleGrade && tgtHandle != HandleFlag {
			return ValidationError{Field: "node:" + tgt.ID, Message: "Flag for Review accepts reason, comments, report, grade, or flag inputs only."}
		}
		if !flagSinkInputSourceIsValid(src, srcHandle, tgtHandle) {
			return ValidationError{Field: "node:" + tgt.ID + "." + tgtHandle, Message: "Invalid source for this Flag for Review input slot."}
		}
	case NodeTypeHumanReviewGate:
		if tgtHandle != HandleComments && tgtHandle != HandleReport && tgtHandle != HandleGrade && tgtHandle != HandleFlag {
			return ValidationError{Field: "node:" + tgt.ID, Message: "Human Review Gate accepts grade (required), comments, report, or flag inputs only."}
		}
		if !gateInputSourceIsValid(src, srcHandle, tgtHandle) {
			return ValidationError{Field: "node:" + tgt.ID + "." + tgtHandle, Message: "Invalid source for this Human Review Gate input slot."}
		}
	case NodeTypeOriginality:
		if tgtHandle != HandleSubmission {
			return ValidationError{Field: "node:" + tgt.ID, Message: "Originality Check accepts a submission input only."}
		}
		if !originalityInputSourceIsValid(src, srcHandle, tgtHandle) {
			return ValidationError{Field: "node:" + tgt.ID, Message: "Submission input must come from a Student Submission node."}
		}
	case NodeTypeScoreAggregator:
		if tgtHandle != HandleGrade {
			return ValidationError{Field: "node:" + tgt.ID, Message: "Score Aggregator accepts grade inputs only."}
		}
		if !aggregatorInputSourceIsValid(src, srcHandle) {
			return ValidationError{Field: "node:" + tgt.ID, Message: "Invalid grade source for Score Aggregator."}
		}
	case NodeTypeSetScore:
		if tgtHandle != HandleGrade {
			return ValidationError{Field: "node:" + tgt.ID, Message: "Set Score accepts a grade input only."}
		}
		if !setScoreInputSourceIsValid(src, srcHandle) {
			return ValidationError{Field: "node:" + tgt.ID, Message: "Set Score input must come from a Conditional Router branch or a grade output."}
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
			if !ok {
				continue
			}
			switch strings.TrimSpace(e.SourceHandle) {
			case HandleContent:
				if isActivityNodeType(src.Type) {
					includeContent = true
				}
			case HandleRubric:
				if isActivityNodeType(src.Type) || isRubricNodeType(src.Type) {
					includeRubric = true
				}
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
			if src, ok := nodeByID[e.Source]; ok && (isActivityNodeType(src.Type) || isRubricNodeType(src.Type)) {
				includeRubric = true
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
		if src, ok := nodeByID[e.Source]; ok {
			if id := resolveGradeWireSource(g, nodeByID, src.ID); id != "" {
				return id
			}
		}
	}
	for _, n := range g.Nodes {
		if gradeSourceNodeType(n.Type) {
			return n.ID
		}
	}
	return ""
}

func gradeSourceNodeType(nodeType string) bool {
	return nodeType == NodeTypeGrader || nodeType == NodeTypeCriterionGrader || nodeType == NodeTypeAI || nodeType == NodeTypeCodeTestRunner || nodeType == NodeTypeScoreAggregator
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

func criterionIDFromNode(n WorkflowNode) (uuid.UUID, error) {
	if n.Data == nil {
		return uuid.Nil, ValidationError{Field: "node:" + n.ID + ".criterionId", Message: "Select a rubric criterion."}
	}
	raw, ok := n.Data["criterionId"].(string)
	if !ok || strings.TrimSpace(raw) == "" {
		return uuid.Nil, ValidationError{Field: "node:" + n.ID + ".criterionId", Message: "Select a rubric criterion."}
	}
	id, err := uuid.Parse(strings.TrimSpace(raw))
	if err != nil {
		return uuid.Nil, ValidationError{Field: "node:" + n.ID + ".criterionId", Message: "Criterion id must be a valid UUID."}
	}
	return id, nil
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
		if !ok {
			continue
		}
		targetHandle := strings.TrimSpace(e.TargetHandle)
		sourceHandle := strings.TrimSpace(e.SourceHandle)
		switch targetHandle {
		case HandleContent:
			if isActivityNodeType(src.Type) {
				contentItemID = activityAssignmentItemID(src)
			}
		case HandleRubric:
			if isActivityNodeType(src.Type) {
				rubricItemID = activityAssignmentItemID(src)
			} else if isRubricNodeType(src.Type) {
				rubricItemID = rubricWiredAssignmentItemID(src)
			}
		case HandleAIInput:
			if !isAINodeType(promptNode.Type) {
				continue
			}
			switch sourceHandle {
			case HandleContent:
				if isActivityNodeType(src.Type) {
					contentItemID = activityAssignmentItemID(src)
				}
			case HandleRubric:
				if isActivityNodeType(src.Type) {
					rubricItemID = activityAssignmentItemID(src)
				} else if isRubricNodeType(src.Type) {
					rubricItemID = rubricWiredAssignmentItemID(src)
				}
			}
		}
	}
	return contentItemID, rubricItemID
}

func rubricWiredAssignmentItemID(n WorkflowNode) string {
	switch rubricSourceFromNode(n) {
	case RubricSourceLibrary:
		return rubricLibraryAssignmentItemID(n)
	default:
		return ""
	}
}

func findGradeSourceNode(g *WorkflowGraph, nodeByID map[string]WorkflowNode) string {
	for _, e := range g.Edges {
		tgt, ok := nodeByID[e.Target]
		if !ok || tgt.Type != NodeTypeOutput || e.TargetHandle != HandleGrade {
			continue
		}
		if src, ok := nodeByID[e.Source]; ok {
			if id := resolveGradeWireSource(g, nodeByID, src.ID); id != "" {
				return id
			}
			if isConditionalRouterNodeType(src.Type) {
				return src.ID
			}
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
	if isCodeTestRunnerNodeType(gradeSource.Type) || isScoreAggregatorNodeType(gradeSource.Type) {
		commentSource := ""
		for _, e := range g.Edges {
			tgt, ok := nodeByID[e.Target]
			if !ok || tgt.Type != NodeTypeOutput || e.TargetHandle != HandleComments {
				continue
			}
			commentSource = e.Source
		}
		return CompiledWorkflow{GradeSource: gradeSourceID, CommentSource: commentSource}, nil
	}
	if isConditionalRouterNodeType(gradeSource.Type) {
		return CompiledWorkflow{
			GradeSource: gradeSourceID,
			ScoreRequest: ScoreRequest{
				SubmissionText: submissionText,
			},
		}, nil
	}
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
