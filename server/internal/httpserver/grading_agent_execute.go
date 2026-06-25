package httpserver

import (
	"context"
	"strings"

	"github.com/google/uuid"

	"github.com/lextures/lextures/server/internal/models/assignmentrubric"
	gradingagentsvc "github.com/lextures/lextures/server/internal/service/gradingagent"
	aigateway "github.com/lextures/lextures/server/internal/service/aigateway"
	"github.com/lextures/lextures/server/internal/service/codeexecution"
)

type gradingAgentPreviewResult struct {
	Points           float64
	Comment          string
	Confidence       float64
	RubricScores     map[string]float64
	Flagged          *gradingagentsvc.DryRunFlagPreview
	Held             *gradingagentsvc.DryRunHeldPreview
	ModelID          *string
	PromptTokens     *int
	CompletionTokens *int
	CostUSD          *float64
	GradedByAI       bool
}

type gradingAgentExecuteInput struct {
	ModelUser       uuid.UUID
	Submissions     []string
	SubmissionText  string
	UseVision       bool
	VisionImages    []string
	WorkflowGraph   *gradingagentsvc.WorkflowGraph
	GradeSourceID   string
	Compiled        gradingagentsvc.CompiledWorkflow
	ContentModality gradingagentsvc.InputModality
	ContentMarkdown string
	Rubric          *assignmentrubric.RubricDefinition
	MaxPoints       float64
	CourseCode      string
	SubmissionID    uuid.UUID
	InstructorPrompt string
	IncludeAssignmentContent bool
	IncludeRubric   bool
	ConfigModelID   *string
}

// gradingAgentVisionComplexGraphBlocked reports vision submissions on router/flag/gate/aggregator graphs.
func gradingAgentVisionComplexGraphBlocked(useVision bool, g *gradingagentsvc.WorkflowGraph) bool {
	return useVision && g != nil && gradingagentsvc.WorkflowRequiresGraphExecution(g)
}

// gradingAgentUsesWorkflowEngine reports whether a compiled workflow graph should be executed via the engine.
func gradingAgentUsesWorkflowEngine(g *gradingagentsvc.WorkflowGraph) bool {
	return g != nil
}

func (d Deps) executeGradingAgentPreview(
	ctx context.Context,
	svc *gradingagentsvc.Service,
	in gradingAgentExecuteInput,
) (gradingAgentPreviewResult, error) {
	if gradingAgentUsesWorkflowEngine(in.WorkflowGraph) {
		return d.executeGradingAgentWorkflowPreview(ctx, svc, in)
	}
	return d.executeGradingAgentLegacyScore(ctx, svc, in)
}

func (d Deps) executeGradingAgentWorkflowPreview(
	ctx context.Context,
	svc *gradingagentsvc.Service,
	in gradingAgentExecuteInput,
) (gradingAgentPreviewResult, error) {
	modelID := in.Compiled.ScoreRequest.ModelID
	if gradingagentsvc.WorkflowUsesLLM(in.WorkflowGraph) {
		var modelErr error
		modelID, modelErr = d.resolveGraderAgentModelID(ctx, in.ModelUser, modelID, in.ConfigModelID)
		if modelErr != nil {
			return gradingAgentPreviewResult{}, modelErr
		}
		dec, _ := aigateway.Evaluate(
			ctx, d.Pool, d.aiGatewayConfig(), in.ModelUser, nil,
			aigateway.FeatureGraderAgent, modelID,
			aigateway.ContentHash(gradingagentsvc.ContentHashInput(in.Compiled.ScoreRequest.InstructorPrompt, in.SubmissionText)),
		)
		if !dec.Allowed {
			return gradingAgentPreviewResult{}, errGradingAgentAIGatewayBlocked
		}
		if svc.Client == nil {
			return gradingAgentPreviewResult{}, errGradingAgentProviderNotConfigured
		}
	}

	preview, execErr := gradingagentsvc.ExecuteWorkflow(ctx, gradingagentsvc.ExecutionInput{
		Graph:           in.WorkflowGraph,
		Submissions:     in.Submissions,
		InputModality:   in.ContentModality,
		SubmissionID:    in.SubmissionID,
		CourseCode:      in.CourseCode,
		DefaultMarkdown: in.ContentMarkdown,
		DefaultRubric:   in.Rubric,
		MaxPoints:       in.MaxPoints,
		ModelID:         modelID,
		LoadOriginalityReports: d.loadOriginalityReportsForGraderAgent,
		LoadReferenceFile: func(ctx context.Context, courseCode string, fileID uuid.UUID) (string, error) {
			return svc.LoadReferenceFileMarkdown(ctx, courseCode, fileID)
		},
		Runner:     svc,
		CodeRunner: codeexecution.New(),
	})
	if execErr != nil {
		return gradingAgentPreviewResult{}, execErr
	}

	return gradingAgentPreviewFromDryRun(preview, gradingagentsvc.WorkflowUsesLLM(in.WorkflowGraph)), nil
}

func (d Deps) executeGradingAgentLegacyScore(
	ctx context.Context,
	svc *gradingagentsvc.Service,
	in gradingAgentExecuteInput,
) (gradingAgentPreviewResult, error) {
	modelID, modelErr := d.resolveGraderAgentModelID(ctx, in.ModelUser, "", in.ConfigModelID)
	if modelErr != nil {
		return gradingAgentPreviewResult{}, modelErr
	}
	prompt := in.InstructorPrompt
	dec, _ := aigateway.Evaluate(
		ctx, d.Pool, d.aiGatewayConfig(), in.ModelUser, nil,
		aigateway.FeatureGraderAgent, modelID,
		aigateway.ContentHash(gradingagentsvc.ContentHashInput(prompt, in.SubmissionText)),
	)
	if !dec.Allowed {
		return gradingAgentPreviewResult{}, errGradingAgentAIGatewayBlocked
	}
	if svc.Client == nil {
		return gradingAgentPreviewResult{}, errGradingAgentProviderNotConfigured
	}

	scoreReq := d.buildLegacyGradingAgentScoreRequest(in, modelID)
	var result gradingagentsvc.ScoreResult
	var err error
	if in.UseVision {
		result, err = svc.ScoreWithVision(ctx, scoreReq, in.VisionImages)
	} else {
		result, err = svc.Score(ctx, scoreReq)
	}
	if err != nil {
		return gradingAgentPreviewResult{}, err
	}
	return gradingAgentPreviewFromScore(result), nil
}

func (d Deps) buildLegacyGradingAgentScoreRequest(in gradingAgentExecuteInput, modelID string) gradingagentsvc.ScoreRequest {
	scoreReq := gradingagentsvc.ScoreRequest{
		InstructorPrompt:         in.InstructorPrompt,
		IncludeAssignmentContent: in.IncludeAssignmentContent,
		IncludeRubric:            in.IncludeRubric,
		ModelID:                  modelID,
		SubmissionText:           in.SubmissionText,
	}
	if in.Compiled.GradeSource != "" {
		scoreReq = in.Compiled.ScoreRequest
		scoreReq.ModelID = modelID
	}
	scoreReq.AssignmentMarkdown = in.ContentMarkdown
	scoreReq.Rubric = in.Rubric
	scoreReq.MaxPoints = in.MaxPoints
	if in.WorkflowGraph != nil && strings.TrimSpace(in.GradeSourceID) != "" {
		scoreReq.InstructorPrompt = gradingagentsvc.SubstituteWorkflowPromptVariables(
			in.WorkflowGraph,
			in.GradeSourceID,
			scoreReq.InstructorPrompt,
			gradingagentsvc.PromptVariableContext{
				Submissions:     in.Submissions,
				ContentMarkdown: in.ContentMarkdown,
				Rubric:          in.Rubric,
			},
		)
	}
	return scoreReq
}

func gradingAgentPreviewFromDryRun(preview gradingagentsvc.DryRunPreview, gradedByAI bool) gradingAgentPreviewResult {
	return gradingAgentPreviewResult{
		Points:       preview.SuggestedPoints,
		Comment:      preview.Comment,
		Confidence:   preview.Confidence,
		RubricScores: preview.RubricScores,
		Flagged:      preview.Flagged,
		Held:         preview.Held,
		GradedByAI:   gradedByAI,
	}
}

func gradingAgentPreviewFromScore(result gradingagentsvc.ScoreResult) gradingAgentPreviewResult {
	pt := result.PromptTokens
	ct := result.CompletionTokens
	cost := result.CostUSD
	model := result.ModelID
	return gradingAgentPreviewResult{
		Points:           result.Output.TotalPoints,
		Comment:          result.Output.Comment,
		Confidence:       result.Output.Confidence,
		RubricScores:     result.Output.RubricScores,
		ModelID:          &model,
		PromptTokens:     &pt,
		CompletionTokens: &ct,
		CostUSD:          &cost,
		GradedByAI:       true,
	}
}

// Sentinel errors mapped to user-facing failure reasons in the queue handler.
var (
	errGradingAgentAIGatewayBlocked      = gradingAgentExecuteError{"AI processing blocked"}
	errGradingAgentProviderNotConfigured = gradingAgentExecuteError{"AI provider not configured"}
)

type gradingAgentExecuteError struct{ msg string }

func (e gradingAgentExecuteError) Error() string { return e.msg }
