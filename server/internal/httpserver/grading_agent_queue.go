package httpserver

import (
	"context"
	"encoding/json"
	"strings"

	gradingagentrepo "github.com/lextures/lextures/server/internal/repos/gradingagent"
	"github.com/lextures/lextures/server/internal/repos/coursegrades"
	"github.com/lextures/lextures/server/internal/repos/coursemoduleassignments"
	"github.com/lextures/lextures/server/internal/repos/moduleassignmentsubmissions"
	"github.com/lextures/lextures/server/internal/gradingagentqueue"
	gradingagentsvc "github.com/lextures/lextures/server/internal/service/gradingagent"
	aigateway "github.com/lextures/lextures/server/internal/service/aigateway"
)

// HandleGradingAgentQueueMessage grades one submission and writes a provisional grade.
func (d Deps) HandleGradingAgentQueueMessage(ctx context.Context, msg gradingagentqueue.QueueMessage) error {
	if d.Pool == nil {
		return nil
	}
	cfg, err := gradingagentrepo.GetConfigByItem(ctx, d.Pool, msg.ItemID)
	if err != nil || cfg == nil {
		return d.failGradingAgentItem(ctx, msg, "config not found")
	}
	assignRow, err := coursemoduleassignments.GetForCourseItem(ctx, d.Pool, msg.CourseID, msg.ItemID)
	if err != nil || assignRow == nil {
		return d.failGradingAgentItem(ctx, msg, "assignment not found")
	}
	subRow, err := moduleassignmentsubmissions.GetByIDForCourse(ctx, d.Pool, msg.CourseID, msg.SubmissionID)
	if err != nil || subRow == nil {
		return d.failGradingAgentItem(ctx, msg, "submission not found")
	}
	run, err := gradingagentrepo.GetRun(ctx, d.Pool, msg.RunID)
	if err != nil || run == nil {
		return d.failGradingAgentItem(ctx, msg, "run not found")
	}
	if run.Scope == gradingagentrepo.RunScopeUngraded {
		cell, cellErr := coursegrades.GetCell(ctx, d.Pool, msg.CourseID, subRow.SubmittedBy, msg.ItemID)
		if cellErr == nil && cell != nil && cell.PointsEarned != nil {
			_, _ = gradingagentrepo.InsertResult(ctx, d.Pool, gradingagentrepo.InsertResultInput{
				RunID: &msg.RunID, ConfigID: cfg.ID, SubmissionID: msg.SubmissionID,
				Status: gradingagentrepo.ItemSkipped, Error: gradingAgentStrPtr("already graded"),
			})
			return gradingagentrepo.IncrementRunProgress(ctx, d.Pool, msg.RunID, false)
		}
	}
	svc := d.gradingAgentService()
	if subRow.AttachmentFileID == nil {
		return d.failGradingAgentItem(ctx, msg, "submission has no file attachment")
	}
	submissionText, err := svc.LoadSubmissionTextForSubmission(ctx, msg.CourseCode, subRow)
	if err != nil {
		return d.failGradingAgentItem(ctx, msg, err.Error())
	}
	modelUser := cfg.CreatedBy
	if run.InitiatedBy != nil {
		modelUser = *run.InitiatedBy
	}
	modelID, modelErr := d.resolveGraderAgentModelID(ctx, modelUser, "", cfg.ModelID)
	if modelErr != nil {
		return d.failGradingAgentItem(ctx, msg, modelErr.Error())
	}
	dec, _ := aigateway.Evaluate(ctx, d.Pool, d.aiGatewayConfig(), modelUser, nil, aigateway.FeatureGraderAgent, modelID, aigateway.ContentHash(gradingagentsvc.ContentHashInput(cfg.Prompt, submissionText)))
	if !dec.Allowed {
		return d.failGradingAgentItem(ctx, msg, "AI processing blocked")
	}
	if svc.Client == nil {
		return d.failGradingAgentItem(ctx, msg, "AI provider not configured")
	}
	rubric, _ := gradingagentsvc.ParseAssignmentRubric(assignRow)
	result, err := svc.Score(ctx, gradingagentsvc.ScoreRequest{
		InstructorPrompt:         cfg.Prompt,
		IncludeAssignmentContent: cfg.IncludeAssignmentContent,
		IncludeRubric:            cfg.IncludeRubric,
		ModelID:                  modelID,
		AssignmentMarkdown:       assignRow.Markdown,
		Rubric:                   rubric,
		MaxPoints:                gradingagentsvc.MaxPointsFromAssignment(assignRow),
		SubmissionText:           submissionText,
	})
	if err != nil {
		return d.failGradingAgentItem(ctx, msg, err.Error())
	}
	comment := result.Output.Comment
	conf := result.Output.Confidence
	pt := result.PromptTokens
	ct := result.CompletionTokens
	cost := result.CostUSD
	model := result.ModelID
	pts := result.Output.TotalPoints
	var rubricJSON []byte
	if len(result.Output.RubricScores) > 0 {
		rubricJSON, _ = json.Marshal(result.Output.RubricScores)
	}
	posting := "manual"
	if strings.TrimSpace(assignRow.PostingPolicy) == "automatic" && cfg.PostPolicy == "auto_post" {
		posting = "automatic"
	}
	gradedByAI := true
	if err := coursegrades.UpsertCellWithFlags(ctx, d.Pool, msg.CourseID, subRow.SubmittedBy, msg.ItemID, pts, rubricJSON, &comment, posting, gradedByAI); err != nil {
		return d.failGradingAgentItem(ctx, msg, "failed to write grade")
	}
	_, _ = gradingagentrepo.InsertResult(ctx, d.Pool, gradingagentrepo.InsertResultInput{
		RunID: &msg.RunID, ConfigID: cfg.ID, SubmissionID: msg.SubmissionID,
		SuggestedPoints: &pts, Comment: &comment, Confidence: &conf,
		Status: gradingagentrepo.ItemApplied, ModelID: &model,
		PromptTokens: &pt, CompletionTokens: &ct, CostUSD: &cost,
	})
	return gradingagentrepo.IncrementRunProgress(ctx, d.Pool, msg.RunID, false)
}

func (d Deps) failGradingAgentItem(ctx context.Context, msg gradingagentqueue.QueueMessage, reason string) error {
	_, _ = gradingagentrepo.InsertResult(ctx, d.Pool, gradingagentrepo.InsertResultInput{
		RunID: &msg.RunID, ConfigID: msg.ConfigID, SubmissionID: msg.SubmissionID,
		Status: gradingagentrepo.ItemFailed, Error: &reason,
	})
	_ = gradingagentrepo.IncrementRunProgress(ctx, d.Pool, msg.RunID, true)
	return nil
}

func gradingAgentStrPtr(s string) *string { return &s }