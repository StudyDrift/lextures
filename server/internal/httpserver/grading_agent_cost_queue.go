package httpserver

import (
	"context"

	"github.com/google/uuid"

	"github.com/lextures/lextures/server/internal/gradingagentqueue"
	gradingagentrepo "github.com/lextures/lextures/server/internal/repos/gradingagent"
	aigateway "github.com/lextures/lextures/server/internal/service/aigateway"
	"github.com/lextures/lextures/server/internal/service/openrouter"
)

func (d Deps) gradingAgentRunOverBudget(ctx context.Context, run *gradingagentrepo.RunRow) (bool, error) {
	if run == nil || run.BudgetUSD == nil || *run.BudgetUSD <= 0 {
		return false, nil
	}
	spent, err := gradingagentrepo.SumRunCostUSD(ctx, d.Pool, run.ID)
	if err != nil {
		return false, err
	}
	return spent >= *run.BudgetUSD, nil
}

func (d Deps) skipGradingAgentBudgetExceeded(ctx context.Context, msg gradingagentqueue.QueueMessage, configID uuid.UUID) error {
	reason := "budget exceeded"
	_, _ = gradingagentrepo.InsertResult(ctx, d.Pool, gradingagentrepo.InsertResultInput{
		RunID: &msg.RunID, ConfigID: configID, SubmissionID: msg.SubmissionID,
		Status: gradingagentrepo.ItemSkipped, Error: &reason,
	})
	_ = gradingagentrepo.MarkRunBudgetExceeded(ctx, d.Pool, msg.RunID)
	return gradingagentrepo.IncrementRunProgress(ctx, d.Pool, msg.RunID, false)
}

func (d Deps) recordGradingAgentQueueUsage(
	ctx context.Context,
	modelUser uuid.UUID,
	courseCode string,
	courseID uuid.UUID,
	preview gradingAgentPreviewResult,
	succeeded bool,
) {
	if preview.ModelID == nil || preview.PromptTokens == nil {
		return
	}
	completion := 0
	if preview.CompletionTokens != nil {
		completion = *preview.CompletionTokens
	}
	cost := 0.0
	if preview.CostUSD != nil {
		cost = *preview.CostUSD
	}
	d.recordAIUsage(ctx, AIUsageMeta{
		UserID: modelUser, CourseID: &courseID, CourseCode: courseCode,
		Feature: aigateway.FeatureGraderAgent, Model: *preview.ModelID,
	}, openrouter.UsageInfo{
		PromptTokens:     *preview.PromptTokens,
		CompletionTokens: completion,
		TotalTokens:      *preview.PromptTokens + completion,
		CostUSD:          cost,
	}, succeeded)
}
