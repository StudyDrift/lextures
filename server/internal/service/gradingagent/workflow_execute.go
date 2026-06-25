package gradingagent

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/lextures/lextures/server/internal/models/assignmentrubric"
)

// ExecutionEvent is streamed while a workflow executes (dry-run WS or live batch grading).
type ExecutionEvent struct {
	Type            string         `json:"type"`
	NodeID          string         `json:"nodeId,omitempty"`
	NodeType        string         `json:"nodeType,omitempty"`
	NodeLabel       string         `json:"nodeLabel,omitempty"`
	Status          string         `json:"status,omitempty"`
	Message         string         `json:"message,omitempty"`
	Level           string         `json:"level,omitempty"`
	CompiledPrompt       string         `json:"compiledPrompt,omitempty"`
	CompiledSystemPrompt string         `json:"compiledSystemPrompt,omitempty"`
	CompiledInput        string         `json:"compiledInput,omitempty"`
	CompiledOutput       string         `json:"compiledOutput,omitempty"`
	Result          *DryRunPreview `json:"result,omitempty"`
}

// DryRunFlagPreview is the assembled flag-for-review preview from a flag sink.
type DryRunFlagPreview struct {
	Reason   string `json:"reason"`
	Queue    string `json:"queue"`
	Priority string `json:"priority"`
}

// DryRunHeldPreview is the hold decision from a Human Review Gate.
type DryRunHeldPreview struct {
	WouldHold       bool    `json:"wouldHold"`
	Mode            string  `json:"mode"`
	Reason          string  `json:"reason"`
	Queue           string  `json:"queue"`
	ConfidenceFloor float64 `json:"confidenceFloor,omitempty"`
}

// DryRunPreview is the assembled grade preview from the output node.
type DryRunPreview struct {
	SuggestedPoints  float64            `json:"suggestedPoints"`
	RubricScores     map[string]float64 `json:"rubricScores,omitempty"`
	Comment          string             `json:"comment"`
	Confidence       float64            `json:"confidence"`
	PromptTokens     int                `json:"promptTokens,omitempty"`
	CompletionTokens int                `json:"completionTokens,omitempty"`
	Flagged          *DryRunFlagPreview `json:"flagged,omitempty"`
	Held             *DryRunHeldPreview `json:"held,omitempty"`
}

// ActivitySource resolves assignment content and rubric for an activity node.
type ActivitySource func(assignmentItemID string) (markdown string, rubric *assignmentrubric.RubricDefinition, err error)

// DryRunRunner executes LLM calls during a workflow dry run.
type DryRunRunner interface {
	Score(ctx context.Context, req ScoreRequest) (ScoreResult, error)
	RunPrompt(ctx context.Context, modelID, systemPrompt, prompt, input string, jsonMode bool) (text string, promptTokens, completionTokens int, err error)
}

// ExecutionInput configures a step-by-step workflow execution (dry-run or live).
type ExecutionInput struct {
	Graph                  *WorkflowGraph
	Submissions            []string
	InputModality          InputModality
	SubmissionID           uuid.UUID
	IsLate                 bool
	DefaultMarkdown        string
	DefaultRubric          *assignmentrubric.RubricDefinition
	MaxPoints              float64
	ModelID                string
	ResolveActivity        ActivitySource
	LoadOriginalityReports func(ctx context.Context, submissionID uuid.UUID) ([]OriginalityReportRow, error)
	LoadReferenceFile      func(ctx context.Context, courseCode string, fileID uuid.UUID) (string, error)
	CourseCode             string
	Runner                 DryRunRunner
	CodeRunner             CodeTestRunner
	Emit                   func(ExecutionEvent)
}

type slotValue struct {
	text   string
	grade  *GradeOutput
	rubric *assignmentrubric.RubricDefinition
	score  *float64
	flag   *bool
}

type executionState struct {
	values     map[string]slotValue // key: nodeID:handle
	edgeActive map[string]bool      // edge id -> active for branch routing
}

func (s *executionState) set(nodeID, handle string, v slotValue) {
	if s.values == nil {
		s.values = make(map[string]slotValue)
	}
	s.values[nodeID+":"+handle] = v
}

func (s *executionState) get(nodeID, handle string) (slotValue, bool) {
	if s.values == nil {
		return slotValue{}, false
	}
	v, ok := s.values[nodeID+":"+handle]
	return v, ok
}

func newExecutionState(g *WorkflowGraph) *executionState {
	edgeActive := make(map[string]bool, len(g.Edges))
	for _, e := range g.Edges {
		edgeActive[e.ID] = true
	}
	return &executionState{values: make(map[string]slotValue), edgeActive: edgeActive}
}

func (s *executionState) deactivateRouterBranch(g *WorkflowGraph, routerID, handle string) {
	for _, e := range g.Edges {
		if e.Source == routerID && strings.TrimSpace(e.SourceHandle) == handle {
			s.edgeActive[e.ID] = false
		}
	}
}

func nodeHasActiveInput(g *WorkflowGraph, nodeID string, nodeByID map[string]WorkflowNode, state *executionState) bool {
	n, ok := nodeByID[nodeID]
	if !ok {
		return false
	}
	if isWorkflowSourceNode(n.Type) {
		return true
	}

	activeInbound := 0
	valuedInbound := 0
	hasAnyInbound := false
	for _, e := range g.Edges {
		if e.Target != nodeID {
			continue
		}
		hasAnyInbound = true
		if !state.edgeActive[e.ID] {
			continue
		}
		activeInbound++
		srcHandle := strings.TrimSpace(e.SourceHandle)
		if _, ok := state.get(e.Source, srcHandle); ok {
			valuedInbound++
		}
	}

	if activeInbound > 0 {
		return valuedInbound > 0
	}
	if hasAnyInbound {
		return false
	}
	return nodeRunsWithoutWiredInput(n.Type)
}

func nodeRunsWithoutWiredInput(nodeType string) bool {
	switch nodeType {
	case NodeTypeGrader, NodeTypeCriterionGrader, NodeTypeAI, NodeTypeCodeTestRunner:
		return true
	default:
		return false
	}
}

func gatherRouterInput(g *WorkflowGraph, routerID string, nodeByID map[string]WorkflowNode, state *executionState) (slotValue, bool) {
	for _, e := range g.Edges {
		if e.Target != routerID || strings.TrimSpace(e.TargetHandle) != HandleAIInput {
			continue
		}
		if !state.edgeActive[e.ID] {
			continue
		}
		srcHandle := strings.TrimSpace(e.SourceHandle)
		if v, ok := state.get(e.Source, srcHandle); ok {
			return v, true
		}
	}
	return slotValue{}, false
}

func buildPredicateContext(in ExecutionInput, input slotValue) PredicateEvalContext {
	ctx := PredicateEvalContext{
		SubmissionText: JoinSubmissions(in.Submissions),
		IsLate:         in.IsLate,
	}
	if input.grade != nil {
		ctx.InputGrade = input.grade
	}
	if input.score != nil {
		ctx.InputScore = input.score
		ctx.OriginalityScore = input.score
	}
	return ctx
}

// NodeDisplayLabel returns the instructor-facing label for a workflow node.
func NodeDisplayLabel(data map[string]any, nodeType string) string {
	return workflowNodeDisplayLabel(data, nodeType)
}

// TopologicalNodeOrder returns node ids in dependency order (sources before sinks).
func TopologicalNodeOrder(g *WorkflowGraph) ([]string, error) {
	if g == nil {
		return nil, ValidationError{Field: "workflowGraph", Message: "Workflow graph is required."}
	}
	nodeByID := make(map[string]WorkflowNode, len(g.Nodes))
	inDegree := make(map[string]int, len(g.Nodes))
	adj := make(map[string][]string, len(g.Nodes))
	for _, n := range g.Nodes {
		nodeByID[n.ID] = n
		inDegree[n.ID] = 0
	}
	for _, e := range g.Edges {
		if _, ok := nodeByID[e.Source]; !ok {
			continue
		}
		if _, ok := nodeByID[e.Target]; !ok {
			continue
		}
		adj[e.Source] = append(adj[e.Source], e.Target)
		inDegree[e.Target]++
	}
	queue := make([]string, 0, len(g.Nodes))
	for id := range inDegree {
		if inDegree[id] == 0 {
			queue = append(queue, id)
		}
	}
	order := make([]string, 0, len(g.Nodes))
	for len(queue) > 0 {
		id := queue[0]
		queue = queue[1:]
		order = append(order, id)
		for _, next := range adj[id] {
			inDegree[next]--
			if inDegree[next] == 0 {
				queue = append(queue, next)
			}
		}
	}
	if len(order) != len(g.Nodes) {
		return nil, ValidationError{Field: "workflowGraph.edges", Message: "Workflow graph must be acyclic."}
	}
	return order, nil
}

// ExecuteWorkflow walks the graph node by node, streaming execution events when Emit is set.
func ExecuteWorkflow(ctx context.Context, in ExecutionInput) (DryRunPreview, error) {
	if err := ValidateWorkflowGraph(in.Graph); err != nil {
		return DryRunPreview{}, err
	}
	if in.Runner == nil && workflowUsesLLM(in.Graph) {
		return DryRunPreview{}, fmt.Errorf("dry run runner not configured")
	}
	if !workflowUsesLLM(in.Graph) && workflowUsesCodeRunner(in.Graph) && in.CodeRunner == nil {
		return DryRunPreview{}, fmt.Errorf("code execution service is not configured")
	}
	emit := in.Emit
	if emit == nil {
		emit = func(ExecutionEvent) {}
	}

	nodeByID := make(map[string]WorkflowNode, len(in.Graph.Nodes))
	for _, n := range in.Graph.Nodes {
		nodeByID[n.ID] = n
	}
	order, err := TopologicalNodeOrder(in.Graph)
	if err != nil {
		return DryRunPreview{}, err
	}

	state := newExecutionState(in.Graph)
	var preview DryRunPreview
	var totalPromptTokens, totalCompletionTokens int

	for _, nodeID := range order {
		node, ok := nodeByID[nodeID]
		if !ok {
			continue
		}
		label := NodeDisplayLabel(node.Data, node.Type)

		if !nodeHasActiveInput(in.Graph, node.ID, nodeByID, state) {
			emit(ExecutionEvent{Type: "node_start", NodeID: node.ID, NodeType: node.Type, NodeLabel: label})
			emit(ExecutionEvent{
				Type: "log", Level: "info",
				Message: fmt.Sprintf("[%s] Skipped (inactive branch).", label),
			})
			emit(ExecutionEvent{Type: "node_complete", NodeID: node.ID, NodeType: node.Type, Status: "skipped"})
			continue
		}

		emit(ExecutionEvent{Type: "node_start", NodeID: node.ID, NodeType: node.Type, NodeLabel: label})

		var compiledPrompt, compiledSystemPrompt, compiledInput, compiledOutput string
		switch node.Type {
		case NodeTypeStudentSubmission:
			text := JoinSubmissions(in.Submissions)
			state.set(node.ID, HandleSubmission, slotValue{text: text})
			modality := in.InputModality.ModalityLogLabel()
			if modality == "unreadable" || modality == "" {
				modality = "file"
			}
			emit(ExecutionEvent{
				Type: "log", Level: "info",
				Message: fmt.Sprintf("[%s] Input modality: %s; loaded %d part(s) (%d characters).", label, modality, len(in.Submissions), len(text)),
			})
		case NodeTypeActivity:
			itemID := activityAssignmentItemID(node)
			markdown, rubric, loadErr := in.resolveActivity(itemID)
			if loadErr != nil {
				emit(ExecutionEvent{Type: "node_complete", NodeID: node.ID, Status: "error"})
				return DryRunPreview{}, loadErr
			}
			state.set(node.ID, HandleContent, slotValue{text: markdown})
			state.set(node.ID, HandleRubric, slotValue{text: formatRubricVariableText(rubric), rubric: rubric})
			emit(ExecutionEvent{
				Type: "log", Level: "info",
				Message: fmt.Sprintf("[%s] Loaded assignment content (%d chars) and rubric.", label, len(markdown)),
			})
		case NodeTypeRubric:
			rubric, loadErr := in.LoadRubricDefinition(node)
			if loadErr != nil {
				emit(ExecutionEvent{Type: "node_complete", NodeID: node.ID, Status: "error"})
				return DryRunPreview{}, loadErr
			}
			state.set(node.ID, HandleRubric, slotValue{text: formatRubricVariableText(rubric), rubric: rubric})
			emit(ExecutionEvent{
				Type: "log", Level: "info",
				Message: fmt.Sprintf("[%s] Loaded %d criteria.", label, len(rubric.Criteria)),
			})
		case NodeTypeReference:
			text, truncated, loadErr := in.LoadReferenceText(ctx, node)
			if loadErr != nil {
				emit(ExecutionEvent{Type: "node_complete", NodeID: node.ID, Status: "error"})
				return DryRunPreview{}, loadErr
			}
			state.set(node.ID, HandleReference, slotValue{text: text})
			modeLabel := referenceTrustedLabel(referenceModeFromNode(node))
			logMsg := fmt.Sprintf("[%s] Loaded %d chars of reference.", modeLabel, len(text))
			if truncated {
				logMsg += " (truncated)"
			}
			emit(ExecutionEvent{Type: "log", Level: "info", Message: logMsg})
		case NodeTypeAI:
			prompt := graderPrompt(node)
			inputText := gatherAIInput(in.Graph, node.ID, nodeByID, state)
			promptCtx := buildPromptContext(in.Graph, node.ID, nodeByID, state, in.Submissions, in.DefaultMarkdown, in.DefaultRubric)
			prompt = SubstituteWorkflowPromptVariables(in.Graph, node.ID, prompt, promptCtx)
			outputFormat := AIOutputFormatForNode(in.Graph, node.ID)
			rubricForOutput := resolveAIRubric(in.Graph, node.ID, nodeByID, state, in.DefaultRubric)
			systemPrompt := BuildAISystemPrompt(outputFormat, rubricForOutput, in.MaxPoints)
			if in.Runner == nil {
				emit(ExecutionEvent{Type: "node_complete", NodeID: node.ID, Status: "error"})
				return DryRunPreview{}, fmt.Errorf("dry run runner not configured")
			}
			out, pt, ct, runErr := in.Runner.RunPrompt(ctx, in.ModelID, systemPrompt, prompt, inputText, true)
			if runErr != nil {
				emit(ExecutionEvent{Type: "log", Level: "error", Message: fmt.Sprintf("[%s] %s", label, UserFacingScoreError(runErr))})
				emit(ExecutionEvent{Type: "node_complete", NodeID: node.ID, Status: "error"})
				return DryRunPreview{}, runErr
			}
			grade, parseErr := ParseAIOutput(out, outputFormat, rubricForOutput, in.MaxPoints)
			if parseErr != nil {
				emit(ExecutionEvent{Type: "log", Level: "error", Message: fmt.Sprintf("[%s] %s", label, UserFacingScoreError(parseErr))})
				emit(ExecutionEvent{Type: "node_complete", NodeID: node.ID, Status: "error"})
				return DryRunPreview{}, parseErr
			}
			totalPromptTokens += pt
			totalCompletionTokens += ct
			state.set(node.ID, HandleAIOutput, slotValue{text: out, grade: &grade})
			compiledPrompt = prompt
			compiledSystemPrompt = systemPrompt
			compiledInput = inputText
			compiledOutput = out
			emit(ExecutionEvent{
				Type: "log", Level: "info",
				Message: fmt.Sprintf("[%s] AI output: %s", label, truncateLog(out, 240)),
			})
		case NodeTypeGrader:
			req, buildErr := buildGraderScoreRequest(in, node, nodeByID, state)
			if buildErr != nil {
				emit(ExecutionEvent{Type: "node_complete", NodeID: node.ID, Status: "error"})
				return DryRunPreview{}, buildErr
			}
			if in.Runner == nil {
				emit(ExecutionEvent{Type: "node_complete", NodeID: node.ID, Status: "error"})
				return DryRunPreview{}, fmt.Errorf("dry run runner not configured")
			}
			result, scoreErr := in.Runner.Score(ctx, req)
			if scoreErr != nil {
				emit(ExecutionEvent{Type: "log", Level: "error", Message: fmt.Sprintf("[%s] %s", label, UserFacingScoreError(scoreErr))})
				emit(ExecutionEvent{Type: "node_complete", NodeID: node.ID, Status: "error"})
				return DryRunPreview{}, scoreErr
			}
			totalPromptTokens += result.PromptTokens
			totalCompletionTokens += result.CompletionTokens
			grade := result.Output
			state.set(node.ID, HandleGrade, slotValue{grade: &grade})
			state.set(node.ID, HandleComments, slotValue{text: grade.Comment})
			emit(ExecutionEvent{
				Type: "log", Level: "info",
				Message: fmt.Sprintf("[%s] Suggested score %.2f (confidence %.0f%%).", label, grade.TotalPoints, grade.Confidence*100),
			})
		case NodeTypeCriterionGrader:
			criterionID, cidErr := criterionIDFromNode(node)
			if cidErr != nil {
				emit(ExecutionEvent{Type: "node_complete", NodeID: node.ID, Status: "error"})
				return DryRunPreview{}, cidErr
			}
			req, buildErr := buildGraderScoreRequest(in, node, nodeByID, state)
			if buildErr != nil {
				emit(ExecutionEvent{Type: "node_complete", NodeID: node.ID, Status: "error"})
				return DryRunPreview{}, buildErr
			}
			criterion, critErr := findRubricCriterion(req.Rubric, criterionID)
			if critErr != nil {
				emit(ExecutionEvent{Type: "log", Level: "error", Message: fmt.Sprintf("[%s] %s", label, critErr.Error())})
				emit(ExecutionEvent{Type: "node_complete", NodeID: node.ID, Status: "error"})
				return DryRunPreview{}, ValidationError{Field: "node:" + node.ID + ".criterionId", Message: critErr.Error()}
			}
			if in.Runner == nil {
				emit(ExecutionEvent{Type: "node_complete", NodeID: node.ID, Status: "error"})
				return DryRunPreview{}, fmt.Errorf("dry run runner not configured")
			}
			prompt := req.InstructorPrompt
			systemPrompt := BuildCriterionSystemPrompt(criterion)
			userMessage := BuildCriterionUserMessage(
				prompt,
				req.IncludeAssignmentContent,
				req.IncludeRubric,
				req.AssignmentMarkdown,
				req.Rubric,
				criterion,
				req.SubmissionText,
			)
			out, pt, ct, runErr := in.Runner.RunPrompt(ctx, req.ModelID, systemPrompt, userMessage, "", true)
			if runErr != nil {
				emit(ExecutionEvent{Type: "log", Level: "error", Message: fmt.Sprintf("[%s] %s", label, UserFacingScoreError(runErr))})
				emit(ExecutionEvent{Type: "node_complete", NodeID: node.ID, Status: "error"})
				return DryRunPreview{}, runErr
			}
			grade, parseErr := ParseSingleCriterionOutput(out, req.Rubric, criterionID)
			if parseErr != nil {
				emit(ExecutionEvent{Type: "log", Level: "error", Message: fmt.Sprintf("[%s] %s", label, UserFacingScoreError(parseErr))})
				emit(ExecutionEvent{Type: "node_complete", NodeID: node.ID, Status: "error"})
				return DryRunPreview{}, parseErr
			}
			totalPromptTokens += pt
			totalCompletionTokens += ct
			state.set(node.ID, HandleGrade, slotValue{grade: &grade})
			state.set(node.ID, HandleComments, slotValue{text: grade.Comment})
			compiledPrompt = prompt
			compiledSystemPrompt = systemPrompt
			compiledInput = userMessage
			compiledOutput = out
			emit(ExecutionEvent{
				Type: "log", Level: "info",
				Message: fmt.Sprintf("[%s] Criterion %q: %.2f (confidence %.0f%%).", label, criterion.Title, grade.TotalPoints, grade.Confidence*100),
			})
		case NodeTypeCodeTestRunner:
			if execErr := executeCodeTestRunnerNode(ctx, node, in.Graph, nodeByID, state, in.CodeRunner, emit, label); execErr != nil {
				emit(ExecutionEvent{Type: "log", Level: "error", Message: fmt.Sprintf("[%s] %s", label, UserFacingScoreError(execErr))})
				emit(ExecutionEvent{Type: "node_complete", NodeID: node.ID, Status: "error"})
				return DryRunPreview{}, execErr
			}
		case NodeTypeConditionalRouter:
			cond, condErr := routerConditionFromNode(node)
			if condErr != nil {
				emit(ExecutionEvent{Type: "node_complete", NodeID: node.ID, Status: "error"})
				return DryRunPreview{}, ValidationError{Field: "node:" + node.ID + ".condition", Message: condErr.Error()}
			}
			inputVal, hasInput := gatherRouterInput(in.Graph, node.ID, nodeByID, state)
			if !hasInput {
				emit(ExecutionEvent{Type: "node_complete", NodeID: node.ID, Status: "error"})
				return DryRunPreview{}, ValidationError{Field: "node:" + node.ID, Message: "Router input is required."}
			}
			predCtx := buildPredicateContext(in, inputVal)
			result, evalErr := EvaluateRouterCondition(cond, predCtx)
			if evalErr != nil {
				emit(ExecutionEvent{Type: "log", Level: "error", Message: fmt.Sprintf("[%s] %s", label, evalErr.Error())})
				emit(ExecutionEvent{Type: "node_complete", NodeID: node.ID, Status: "error"})
				return DryRunPreview{}, evalErr
			}
			takenHandle := HandleElse
			untakenHandle := HandleThen
			branchLabel := "else"
			if result {
				takenHandle = HandleThen
				untakenHandle = HandleElse
				branchLabel = "then"
			}
			state.set(node.ID, takenHandle, inputVal)
			state.deactivateRouterBranch(in.Graph, node.ID, untakenHandle)
			emit(ExecutionEvent{
				Type: "log", Level: "info",
				Message: fmt.Sprintf("[%s] Condition %s → %s branch.", label, formatRouterConditionSentence(cond), branchLabel),
			})
		case NodeTypeOriginality:
			if execErr := executeOriginalityNode(ctx, node, in, state, emit, label); execErr != nil {
				emit(ExecutionEvent{Type: "log", Level: "error", Message: fmt.Sprintf("[%s] %s", label, UserFacingScoreError(execErr))})
				emit(ExecutionEvent{Type: "node_complete", NodeID: node.ID, Status: "error"})
				return DryRunPreview{}, execErr
			}
		case NodeTypeScoreAggregator:
			if execErr := executeScoreAggregatorNode(in.Graph, node, nodeByID, state, in.MaxPoints, in.DefaultRubric, emit, label); execErr != nil {
				emit(ExecutionEvent{Type: "log", Level: "error", Message: fmt.Sprintf("[%s] %s", label, execErr.Error())})
				emit(ExecutionEvent{Type: "node_complete", NodeID: node.ID, Status: "error"})
				return DryRunPreview{}, execErr
			}
		case NodeTypeHumanReviewGate:
			gradeVal, gradeErr := gatherGateGrade(in.Graph, node.ID, nodeByID, state)
			if gradeErr != nil {
				emit(ExecutionEvent{Type: "node_complete", NodeID: node.ID, Status: "error"})
				return DryRunPreview{}, gradeErr
			}
			flagTruthy := gatherGateFlagTruthy(in.Graph, node.ID, nodeByID, state)
			mode := gateModeFromNode(node)
			floor := gateConfidenceFloorFromNode(node)
			queue := gateQueueFromNode(node)
			wouldHold, holdReason := EvaluateHoldDecision(mode, floor, gradeVal, flagTruthy)
			state.set(node.ID, HandleGrade, slotValue{grade: gradeVal})
			preview.Held = &DryRunHeldPreview{
				WouldHold:       wouldHold,
				Mode:            string(mode),
				Reason:          holdReason,
				Queue:           queue,
				ConfidenceFloor: floor,
			}
			emit(ExecutionEvent{
				Type: "log", Level: "info",
				Message: fmt.Sprintf("Would hold for review (mode=%s, would-hold=%v)", mode, wouldHold),
			})
			if wouldHold && holdReason != "" {
				emit(ExecutionEvent{Type: "log", Level: "info", Message: holdReason})
			}
		case NodeTypeFlagForReview:
			reason, queue, priority := assembleFlagForReview(in.Graph, node, nodeByID, state, in)
			preview.Flagged = &DryRunFlagPreview{Reason: reason, Queue: queue, Priority: priority}
			preview.PromptTokens = totalPromptTokens
			preview.CompletionTokens = totalCompletionTokens
			emit(ExecutionEvent{Type: "log", Level: "info", Message: "── Flag for Review (dry run — not persisted) ──"})
			emit(ExecutionEvent{Type: "log", Level: "info", Message: fmt.Sprintf("Would flag for review: %s", truncateLog(reason, 400))})
			emit(ExecutionEvent{Type: "log", Level: "info", Message: fmt.Sprintf("Queue: %s · Priority: %s", queue, priority)})
			emit(ExecutionEvent{Type: "result", Result: &preview})
		case NodeTypeOutput:
			if preview.Flagged != nil {
				emit(ExecutionEvent{Type: "log", Level: "info", Message: fmt.Sprintf("[%s] Skipped (flagged on another branch).", label)})
				emit(ExecutionEvent{Type: "node_complete", NodeID: node.ID, NodeType: node.Type, Status: "skipped"})
				continue
			}
			held := preview.Held
			preview, err = assembleOutputPreview(in.Graph, nodeByID, state, in.MaxPoints)
			if err != nil {
				emit(ExecutionEvent{Type: "node_complete", NodeID: node.ID, Status: "error"})
				return DryRunPreview{}, err
			}
			preview.Held = held
			preview.PromptTokens = totalPromptTokens
			preview.CompletionTokens = totalCompletionTokens
			emit(ExecutionEvent{Type: "log", Level: "info", Message: "── Student Grade (dry run — not persisted) ──"})
			emit(ExecutionEvent{Type: "log", Level: "info", Message: fmt.Sprintf("Score: %.2f", preview.SuggestedPoints)})
			if len(preview.RubricScores) > 0 {
				emit(ExecutionEvent{Type: "log", Level: "info", Message: fmt.Sprintf("Rubric: %d criteria scored.", len(preview.RubricScores))})
			}
			if strings.TrimSpace(preview.Comment) != "" {
				emit(ExecutionEvent{Type: "log", Level: "info", Message: fmt.Sprintf("Comment: %s", truncateLog(preview.Comment, 400))})
			} else {
				emit(ExecutionEvent{Type: "log", Level: "info", Message: "Comment: (none)"})
			}
			emit(ExecutionEvent{Type: "result", Result: &preview})
		default:
			emit(ExecutionEvent{Type: "log", Level: "warn", Message: fmt.Sprintf("[%s] Skipped unsupported node type %q.", label, node.Type)})
		}

		emit(ExecutionEvent{
			Type:                 "node_complete",
			NodeID:               node.ID,
			NodeType:             node.Type,
			Status:               "success",
			CompiledPrompt:       compiledPrompt,
			CompiledSystemPrompt: compiledSystemPrompt,
			CompiledInput:        compiledInput,
			CompiledOutput:       compiledOutput,
		})
	}

	return preview, nil
}

func (in ExecutionInput) resolveActivity(itemID string) (string, *assignmentrubric.RubricDefinition, error) {
	if in.ResolveActivity != nil {
		return in.ResolveActivity(itemID)
	}
	return in.DefaultMarkdown, in.DefaultRubric, nil
}

func resolveAIRubric(
	g *WorkflowGraph,
	nodeID string,
	nodeByID map[string]WorkflowNode,
	state *executionState,
	defaultRubric *assignmentrubric.RubricDefinition,
) *assignmentrubric.RubricDefinition {
	for _, e := range g.Edges {
		if e.Target != nodeID || strings.TrimSpace(e.TargetHandle) != HandleAIInput {
			continue
		}
		if strings.TrimSpace(e.SourceHandle) != HandleRubric {
			continue
		}
		src, ok := nodeByID[e.Source]
		if !ok {
			continue
		}
		if v, ok := state.get(src.ID, HandleRubric); ok && v.rubric != nil {
			return v.rubric
		}
	}
	return defaultRubric
}

func gatherAIInput(g *WorkflowGraph, nodeID string, nodeByID map[string]WorkflowNode, state *executionState) string {
	var parts []string
	for _, e := range g.Edges {
		if e.Target != nodeID || strings.TrimSpace(e.TargetHandle) != HandleAIInput {
			continue
		}
		if !state.edgeActive[e.ID] {
			continue
		}
		src, ok := nodeByID[e.Source]
		if !ok {
			continue
		}
		handle := strings.TrimSpace(e.SourceHandle)
		v, ok := state.get(src.ID, handle)
		if !ok {
			continue
		}
		if isOriginalityNodeType(src.Type) {
			block := formatOriginalityTrustedAIBlock(src, handle, v)
			if block != "" {
				parts = append(parts, block)
			}
			continue
		}
		if isReferenceNodeType(src.Type) {
			block := formatReferenceTrustedAIBlock(src, v.text)
			if block != "" {
				parts = append(parts, block)
			}
			continue
		}
		if strings.TrimSpace(v.text) != "" {
			label := NodeDisplayLabel(src.Data, src.Type)
			parts = append(parts, fmt.Sprintf("## %s\n%s", label, v.text))
		}
	}
	return strings.Join(parts, "\n\n")
}

func formatOriginalityTrustedAIBlock(src WorkflowNode, handle string, v slotValue) string {
	metric := originalityMetricFromNode(src)
	switch handle {
	case HandleScore:
		if v.score != nil {
			metricName := "similarity"
			if metric == OriginalityMetricAILikelihood {
				metricName = "AI-likelihood"
			}
			return fmt.Sprintf("## Integrity signal (trusted)\n%s: %.2f", metricName, *v.score)
		}
		if strings.TrimSpace(v.text) != "" {
			return fmt.Sprintf("## Integrity signal (trusted)\n%s", strings.TrimSpace(v.text))
		}
	case HandleReport:
		if strings.TrimSpace(v.text) != "" {
			return fmt.Sprintf("## Integrity report (trusted)\n%s", strings.TrimSpace(v.text))
		}
	}
	return ""
}

func buildPromptContext(
	g *WorkflowGraph,
	nodeID string,
	nodeByID map[string]WorkflowNode,
	state *executionState,
	defaultSubmissions []string,
	defaultMarkdown string,
	defaultRubric *assignmentrubric.RubricDefinition,
) PromptVariableContext {
	ctx := PromptVariableContext{
		Submissions:     defaultSubmissions,
		ContentMarkdown: defaultMarkdown,
		Rubric:          defaultRubric,
		ReferenceTexts:  make(map[string]string),
	}
	for _, e := range g.Edges {
		if e.Target != nodeID {
			continue
		}
		if !state.edgeActive[e.ID] {
			continue
		}
		src, ok := nodeByID[e.Source]
		if !ok {
			continue
		}
		handle := strings.TrimSpace(e.SourceHandle)
		v, ok := state.get(src.ID, handle)
		if !ok {
			continue
		}
		switch handle {
		case HandleContent:
			if v.text != "" {
				ctx.ContentMarkdown = v.text
			}
		case HandleRubric:
			if v.rubric != nil {
				ctx.Rubric = v.rubric
			} else if v.text != "" && ctx.Rubric == nil {
				ctx.ContentMarkdown = strings.TrimSpace(ctx.ContentMarkdown + "\n" + v.text)
			}
		case HandleReference:
			if v.text != "" {
				ctx.ReferenceTexts[src.ID] = v.text
			}
		}
		_ = src
	}
	return ctx
}

func buildGraderScoreRequest(
	in ExecutionInput,
	node WorkflowNode,
	nodeByID map[string]WorkflowNode,
	state *executionState,
) (ScoreRequest, error) {
	submissionText := JoinSubmissions(in.Submissions)
	contentMarkdown := in.DefaultMarkdown
	var rubric = in.DefaultRubric
	includeContent := false
	includeRubric := false

	for _, e := range in.Graph.Edges {
		if e.Target != node.ID {
			continue
		}
		if !state.edgeActive[e.ID] {
			continue
		}
		src, ok := nodeByID[e.Source]
		if !ok {
			continue
		}
		handle := strings.TrimSpace(e.SourceHandle)
		v, ok := state.get(src.ID, handle)
		if !ok {
			continue
		}
		switch strings.TrimSpace(e.TargetHandle) {
		case HandleSubmission:
			if v.text != "" {
				submissionText = v.text
			}
		case HandleContent:
			includeContent = true
			if v.text != "" {
				contentMarkdown = v.text
			}
		case HandleRubric:
			includeRubric = true
			if v.rubric != nil {
				rubric = v.rubric
			}
		}
	}

	prompt := graderPrompt(node)
	promptCtx := buildPromptContext(in.Graph, node.ID, nodeByID, state, in.Submissions, contentMarkdown, rubric)
	prompt = SubstituteWorkflowPromptVariables(in.Graph, node.ID, prompt, promptCtx)
	modelID := graderModelID(node)
	if modelID == "" {
		modelID = in.ModelID
	}
	return ScoreRequest{
		InstructorPrompt:         prompt,
		IncludeAssignmentContent: includeContent,
		IncludeRubric:            includeRubric,
		ModelID:                  modelID,
		AssignmentMarkdown:       contentMarkdown,
		Rubric:                   rubric,
		MaxPoints:                in.MaxPoints,
		SubmissionText:           submissionText,
	}, nil
}

func assembleOutputPreview(
	g *WorkflowGraph,
	nodeByID map[string]WorkflowNode,
	state *executionState,
	maxPoints float64,
) (DryRunPreview, error) {
	var preview DryRunPreview
	for _, e := range g.Edges {
		tgt, ok := nodeByID[e.Target]
		if !ok || tgt.Type != NodeTypeOutput {
			continue
		}
		if !state.edgeActive[e.ID] {
			continue
		}
		src, ok := nodeByID[e.Source]
		if !ok {
			continue
		}
		handle := strings.TrimSpace(e.SourceHandle)
		v, ok := state.get(src.ID, handle)
		if !ok {
			continue
		}
		switch strings.TrimSpace(e.TargetHandle) {
		case HandleGrade:
			if v.grade != nil {
				preview.SuggestedPoints = v.grade.TotalPoints
				preview.RubricScores = v.grade.RubricScores
				preview.Confidence = v.grade.Confidence
				if strings.TrimSpace(preview.Comment) == "" {
					preview.Comment = v.grade.Comment
				}
			} else if strings.TrimSpace(v.text) == "" && predCtxEmptyGrade(v) {
				preview.SuggestedPoints = 0
				preview.Confidence = 1
			} else if strings.TrimSpace(v.text) != "" {
				if pts, err := strconv.ParseFloat(strings.Fields(v.text)[0], 64); err == nil {
					preview.SuggestedPoints = pts
					preview.Confidence = 0.5
				} else if maxPoints > 0 {
					preview.SuggestedPoints = maxPoints
					preview.Confidence = 1
				}
			}
		case HandleComments:
			if strings.TrimSpace(v.text) != "" {
				preview.Comment = v.text
			} else if v.grade != nil && strings.TrimSpace(v.grade.Comment) != "" {
				preview.Comment = v.grade.Comment
			}
		}
	}
	if maxPoints > 0 && preview.SuggestedPoints > maxPoints {
		preview.SuggestedPoints = maxPoints
	}
	if preview.SuggestedPoints < 0 {
		preview.SuggestedPoints = 0
	}
	return preview, nil
}

func truncateLog(s string, max int) string {
	s = strings.TrimSpace(s)
	if len(s) <= max {
		return s
	}
	return s[:max] + "…"
}

func predCtxEmptyGrade(v slotValue) bool {
	return v.grade == nil && strings.TrimSpace(v.text) == ""
}