package gradingagent

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/lextures/lextures/server/internal/models/assignmentrubric"
)

// DryRunEvent is streamed to the client while a workflow dry run executes.
type DryRunEvent struct {
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

// DryRunPreview is the assembled grade preview from the output node.
type DryRunPreview struct {
	SuggestedPoints  float64            `json:"suggestedPoints"`
	RubricScores     map[string]float64 `json:"rubricScores,omitempty"`
	Comment          string             `json:"comment"`
	Confidence       float64            `json:"confidence"`
	PromptTokens     int                `json:"promptTokens,omitempty"`
	CompletionTokens int                `json:"completionTokens,omitempty"`
}

// ActivitySource resolves assignment content and rubric for an activity node.
type ActivitySource func(assignmentItemID string) (markdown string, rubric *assignmentrubric.RubricDefinition, err error)

// DryRunRunner executes LLM calls during a workflow dry run.
type DryRunRunner interface {
	Score(ctx context.Context, req ScoreRequest) (ScoreResult, error)
	RunPrompt(ctx context.Context, modelID, systemPrompt, prompt, input string, jsonMode bool) (text string, promptTokens, completionTokens int, err error)
}

// DryRunExecutionInput configures a step-by-step workflow dry run.
type DryRunExecutionInput struct {
	Graph            *WorkflowGraph
	Submissions      []string
	DefaultMarkdown  string
	DefaultRubric    *assignmentrubric.RubricDefinition
	MaxPoints        float64
	ModelID          string
	ResolveActivity  ActivitySource
	Runner           DryRunRunner
	Emit             func(DryRunEvent)
}

type slotValue struct {
	text   string
	grade  *GradeOutput
	rubric *assignmentrubric.RubricDefinition
}

type executionState struct {
	values map[string]slotValue // key: nodeID:handle
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

// ExecuteWorkflowDryRun walks the graph node by node and streams dry-run events.
func ExecuteWorkflowDryRun(ctx context.Context, in DryRunExecutionInput) (DryRunPreview, error) {
	if err := ValidateWorkflowGraph(in.Graph); err != nil {
		return DryRunPreview{}, err
	}
	if in.Runner == nil {
		return DryRunPreview{}, fmt.Errorf("dry run runner not configured")
	}
	emit := in.Emit
	if emit == nil {
		emit = func(DryRunEvent) {}
	}

	nodeByID := make(map[string]WorkflowNode, len(in.Graph.Nodes))
	for _, n := range in.Graph.Nodes {
		nodeByID[n.ID] = n
	}
	order, err := TopologicalNodeOrder(in.Graph)
	if err != nil {
		return DryRunPreview{}, err
	}

	state := &executionState{}
	var preview DryRunPreview
	var totalPromptTokens, totalCompletionTokens int

	for _, nodeID := range order {
		node, ok := nodeByID[nodeID]
		if !ok {
			continue
		}
		label := NodeDisplayLabel(node.Data, node.Type)
		emit(DryRunEvent{Type: "node_start", NodeID: node.ID, NodeType: node.Type, NodeLabel: label})

		var compiledPrompt, compiledSystemPrompt, compiledInput, compiledOutput string
		switch node.Type {
		case NodeTypeStudentSubmission, NodeTypeSubmission:
			text := JoinSubmissions(in.Submissions)
			state.set(node.ID, HandleSubmission, slotValue{text: text})
			emit(DryRunEvent{
				Type: "log", Level: "info",
				Message: fmt.Sprintf("[%s] Loaded %d submission(s) (%d characters).", label, len(in.Submissions), len(text)),
			})
		case NodeTypeActivity, NodeTypeAssignmentCtx:
			itemID := activityAssignmentItemID(node)
			markdown, rubric, loadErr := in.resolveActivity(itemID)
			if loadErr != nil {
				emit(DryRunEvent{Type: "node_complete", NodeID: node.ID, Status: "error"})
				return DryRunPreview{}, loadErr
			}
			state.set(node.ID, HandleContent, slotValue{text: markdown})
			state.set(node.ID, HandleRubric, slotValue{text: formatRubricVariableText(rubric), rubric: rubric})
			emit(DryRunEvent{
				Type: "log", Level: "info",
				Message: fmt.Sprintf("[%s] Loaded assignment content (%d chars) and rubric.", label, len(markdown)),
			})
		case NodeTypeAI:
			prompt := graderPrompt(node)
			inputText := gatherAIInput(in.Graph, node.ID, nodeByID, state)
			promptCtx := buildPromptContext(in.Graph, node.ID, nodeByID, state, in.Submissions, in.DefaultMarkdown, in.DefaultRubric)
			prompt = SubstituteWorkflowPromptVariables(in.Graph, node.ID, prompt, promptCtx)
			outputFormat := AIOutputFormatForNode(in.Graph, node.ID)
			rubricForOutput := resolveAIRubric(in.Graph, node.ID, nodeByID, state, in.DefaultRubric)
			systemPrompt := BuildAISystemPrompt(outputFormat, rubricForOutput, in.MaxPoints)
			out, pt, ct, runErr := in.Runner.RunPrompt(ctx, in.ModelID, systemPrompt, prompt, inputText, true)
			if runErr != nil {
				emit(DryRunEvent{Type: "log", Level: "error", Message: fmt.Sprintf("[%s] %s", label, UserFacingScoreError(runErr))})
				emit(DryRunEvent{Type: "node_complete", NodeID: node.ID, Status: "error"})
				return DryRunPreview{}, runErr
			}
			grade, parseErr := ParseAIOutput(out, outputFormat, rubricForOutput, in.MaxPoints)
			if parseErr != nil {
				emit(DryRunEvent{Type: "log", Level: "error", Message: fmt.Sprintf("[%s] %s", label, UserFacingScoreError(parseErr))})
				emit(DryRunEvent{Type: "node_complete", NodeID: node.ID, Status: "error"})
				return DryRunPreview{}, parseErr
			}
			totalPromptTokens += pt
			totalCompletionTokens += ct
			state.set(node.ID, HandleAIOutput, slotValue{text: out, grade: &grade})
			compiledPrompt = prompt
			compiledSystemPrompt = systemPrompt
			compiledInput = inputText
			compiledOutput = out
			emit(DryRunEvent{
				Type: "log", Level: "info",
				Message: fmt.Sprintf("[%s] AI output: %s", label, truncateLog(out, 240)),
			})
		case NodeTypeGrader:
			req, buildErr := buildGraderScoreRequest(in, node, nodeByID, state)
			if buildErr != nil {
				emit(DryRunEvent{Type: "node_complete", NodeID: node.ID, Status: "error"})
				return DryRunPreview{}, buildErr
			}
			result, scoreErr := in.Runner.Score(ctx, req)
			if scoreErr != nil {
				emit(DryRunEvent{Type: "log", Level: "error", Message: fmt.Sprintf("[%s] %s", label, UserFacingScoreError(scoreErr))})
				emit(DryRunEvent{Type: "node_complete", NodeID: node.ID, Status: "error"})
				return DryRunPreview{}, scoreErr
			}
			totalPromptTokens += result.PromptTokens
			totalCompletionTokens += result.CompletionTokens
			grade := result.Output
			state.set(node.ID, HandleGrade, slotValue{grade: &grade})
			state.set(node.ID, HandleComments, slotValue{text: grade.Comment})
			emit(DryRunEvent{
				Type: "log", Level: "info",
				Message: fmt.Sprintf("[%s] Suggested score %.2f (confidence %.0f%%).", label, grade.TotalPoints, grade.Confidence*100),
			})
		case NodeTypeOutput:
			preview, err = assembleOutputPreview(in.Graph, nodeByID, state, in.MaxPoints)
			if err != nil {
				emit(DryRunEvent{Type: "node_complete", NodeID: node.ID, Status: "error"})
				return DryRunPreview{}, err
			}
			preview.PromptTokens = totalPromptTokens
			preview.CompletionTokens = totalCompletionTokens
			emit(DryRunEvent{Type: "log", Level: "info", Message: "── Student Grade (dry run — not persisted) ──"})
			emit(DryRunEvent{Type: "log", Level: "info", Message: fmt.Sprintf("Score: %.2f", preview.SuggestedPoints)})
			if len(preview.RubricScores) > 0 {
				emit(DryRunEvent{Type: "log", Level: "info", Message: fmt.Sprintf("Rubric: %d criteria scored.", len(preview.RubricScores))})
			}
			if strings.TrimSpace(preview.Comment) != "" {
				emit(DryRunEvent{Type: "log", Level: "info", Message: fmt.Sprintf("Comment: %s", truncateLog(preview.Comment, 400))})
			} else {
				emit(DryRunEvent{Type: "log", Level: "info", Message: "Comment: (none)"})
			}
			emit(DryRunEvent{Type: "result", Result: &preview})
		default:
			emit(DryRunEvent{Type: "log", Level: "warn", Message: fmt.Sprintf("[%s] Skipped unsupported node type %q.", label, node.Type)})
		}

		emit(DryRunEvent{
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

func (in DryRunExecutionInput) resolveActivity(itemID string) (string, *assignmentrubric.RubricDefinition, error) {
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
		src, ok := nodeByID[e.Source]
		if !ok {
			continue
		}
		handle := strings.TrimSpace(e.SourceHandle)
		if v, ok := state.get(src.ID, handle); ok && strings.TrimSpace(v.text) != "" {
			label := NodeDisplayLabel(src.Data, src.Type)
			parts = append(parts, fmt.Sprintf("## %s\n%s", label, v.text))
		}
	}
	return strings.Join(parts, "\n\n")
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
	}
	for _, e := range g.Edges {
		if e.Target != nodeID {
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
		}
		_ = src
	}
	return ctx
}

func buildGraderScoreRequest(
	in DryRunExecutionInput,
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
		case HandleContext:
			if isActivityNodeType(src.Type) {
				if src.Type == NodeTypeAssignmentCtx {
					includeContent = boolData(src, "includeContent")
					includeRubric = boolData(src, "includeRubric")
				} else {
					includeContent = true
					includeRubric = true
				}
			}
		}
	}

	prompt := graderPrompt(node)
	promptCtx := PromptVariableContext{
		Submissions:     in.Submissions,
		ContentMarkdown: contentMarkdown,
		Rubric:          rubric,
	}
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
			} else if strings.TrimSpace(v.text) != "" {
				if pts, err := strconv.ParseFloat(strings.Fields(v.text)[0], 64); err == nil {
					preview.SuggestedPoints = pts
				}
				preview.Confidence = 0.5
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