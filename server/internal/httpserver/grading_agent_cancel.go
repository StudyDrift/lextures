package httpserver

import (
	"context"

	"github.com/google/uuid"

	gradingagentrepo "github.com/lextures/lextures/server/internal/repos/gradingagent"
	"github.com/lextures/lextures/server/internal/gradingagentqueue"
)

const gradingAgentCancelSkipReason = "run cancelled"

func (d Deps) gradingAgentRunStatus(ctx context.Context, runID uuid.UUID) (string, error) {
	if status, ok := gradingAgentRunStatusFromCache(runID); ok {
		return status, nil
	}
	status, err := gradingagentrepo.GetRunStatus(ctx, d.Pool, runID)
	if err != nil || status == "" {
		return status, err
	}
	gradingAgentRunStatusCacheSet(runID, status)
	return status, nil
}

func (d Deps) skipGradingAgentItemForCancel(ctx context.Context, msg gradingagentqueue.QueueMessage) error {
	reason := gradingAgentCancelSkipReason
	_, _ = gradingagentrepo.InsertResult(ctx, d.Pool, gradingagentrepo.InsertResultInput{
		RunID: &msg.RunID, ConfigID: msg.ConfigID, SubmissionID: msg.SubmissionID,
		Status: gradingagentrepo.ItemSkipped, Error: &reason,
	})
	_ = gradingagentrepo.IncrementRunProgress(ctx, d.Pool, msg.RunID, false)
	return nil
}
