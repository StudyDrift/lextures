package httpserver

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/lextures/lextures/server/internal/models/gradecomment"
	gradingagentrepo "github.com/lextures/lextures/server/internal/repos/gradingagent"
	"github.com/lextures/lextures/server/internal/repos/coursegrades"
	"github.com/lextures/lextures/server/internal/repos/coursemoduleassignments"
	"github.com/lextures/lextures/server/internal/repos/moduleassignmentsubmissions"
	"github.com/lextures/lextures/server/internal/gradingagentqueue"
	gradingagentsvc "github.com/lextures/lextures/server/internal/service/gradingagent"
)

type gradingAgentSuccessInput struct {
	Points           float64
	Comment          string
	Confidence       float64
	RubricJSON       []byte
	ModelID          *string
	PromptTokens     *int
	CompletionTokens *int
	CostUSD          *float64
	GradedByAI       bool
}

type gradingAgentGateHoldInput struct {
	WouldHold bool
	Reason    string
	Queue     string
}

func gradingAgentHoldDecision(
	cfg *gradingagentrepo.ConfigRow,
	confidence float64,
	gate *gradingAgentGateHoldInput,
) (hold bool, reason string, queue string) {
	queue = "default"
	gateHold := false
	gateReason := ""
	if gate != nil {
		gateHold = gate.WouldHold
		gateReason = gate.Reason
		if strings.TrimSpace(gate.Queue) != "" {
			queue = strings.TrimSpace(gate.Queue)
		}
	}
	agentHold, agentReason := gradingagentsvc.EvaluateAgentConfidenceFloorHold(cfg.ConfidenceFloor, confidence)
	hold, reason = gradingagentsvc.ComposeHoldDecisions(gateHold, gateReason, agentHold, agentReason)
	return hold, reason, queue
}

func (d Deps) insertHeldGradingAgentResult(
	ctx context.Context,
	msg gradingagentqueue.QueueMessage,
	cfg *gradingagentrepo.ConfigRow,
	in gradingAgentSuccessInput,
	heldReason string,
	queue string,
) error {
	now := time.Now()
	_, err := gradingagentrepo.InsertResult(ctx, d.Pool, gradingagentrepo.InsertResultInput{
		RunID: &msg.RunID, ConfigID: cfg.ID, SubmissionID: msg.SubmissionID,
		SuggestedPoints: &in.Points, Comment: &in.Comment, Confidence: &in.Confidence,
		Status: gradingagentrepo.ItemSuggested, HeldReason: &heldReason,
		HeldAt: &now, HeldQueue: &queue, ModelID: in.ModelID,
		PromptTokens: in.PromptTokens, CompletionTokens: in.CompletionTokens, CostUSD: in.CostUSD,
	})
	if err != nil {
		return err
	}
	return gradingagentrepo.IncrementRunProgress(ctx, d.Pool, msg.RunID, false)
}

func (d Deps) finishGradingAgentSuccess(
	ctx context.Context,
	msg gradingagentqueue.QueueMessage,
	cfg *gradingagentrepo.ConfigRow,
	assignRow *coursemoduleassignments.CourseItemAssignmentRow,
	subRow *moduleassignmentsubmissions.SubmissionRow,
	run *gradingagentrepo.RunRow,
	in gradingAgentSuccessInput,
) error {
	if run != nil && run.Mode == gradingagentrepo.RunModeSuggest {
		now := time.Now()
		reason := "Suggest-only run"
		_, err := gradingagentrepo.InsertResult(ctx, d.Pool, gradingagentrepo.InsertResultInput{
			RunID: &msg.RunID, ConfigID: cfg.ID, SubmissionID: msg.SubmissionID,
			SuggestedPoints: &in.Points, Comment: &in.Comment, Confidence: &in.Confidence,
			Status: gradingagentrepo.ItemSuggested, HeldReason: &reason,
			HeldAt: &now, ModelID: in.ModelID, PromptTokens: in.PromptTokens,
			CompletionTokens: in.CompletionTokens, CostUSD: in.CostUSD,
		})
		if err != nil {
			return err
		}
		return gradingagentrepo.IncrementRunProgress(ctx, d.Pool, msg.RunID, false)
	}

	if hold, reason, queue := gradingAgentHoldDecision(cfg, in.Confidence, nil); hold {
		return d.insertHeldGradingAgentResult(ctx, msg, cfg, in, reason, queue)
	}

	posting := gradingAgentCellPosting(assignRow.PostingPolicy, cfg.PostPolicy)
	_, commentJSON, flatComment, _ := gradecomment.Append(nil, gradecomment.Comment{
		DisplayName: "Grading agent",
		Body:        in.Comment,
		Source:      "lextures",
	})
	if err := coursegrades.UpsertCellWithFlags(
		ctx, d.Pool, msg.CourseID, subRow.SubmittedBy, msg.ItemID,
		in.Points, in.RubricJSON, flatComment, commentJSON, posting, in.GradedByAI,
	); err != nil {
		return err
	}
	_, err := gradingagentrepo.InsertResult(ctx, d.Pool, gradingagentrepo.InsertResultInput{
		RunID: &msg.RunID, ConfigID: cfg.ID, SubmissionID: msg.SubmissionID,
		SuggestedPoints: &in.Points, Comment: &in.Comment, Confidence: &in.Confidence,
		Status: gradingagentrepo.ItemApplied, ModelID: in.ModelID,
		PromptTokens: in.PromptTokens, CompletionTokens: in.CompletionTokens, CostUSD: in.CostUSD,
	})
	if err != nil {
		return err
	}
	return gradingagentrepo.IncrementRunProgress(ctx, d.Pool, msg.RunID, false)
}

func (d Deps) persistGradingAgentPreview(
	ctx context.Context,
	msg gradingagentqueue.QueueMessage,
	cfg *gradingagentrepo.ConfigRow,
	assignRow *coursemoduleassignments.CourseItemAssignmentRow,
	subRow *moduleassignmentsubmissions.SubmissionRow,
	run *gradingagentrepo.RunRow,
	preview gradingAgentPreviewResult,
) error {
	if preview.Flagged != nil {
		reason := preview.Flagged.Reason
		priority := preview.Flagged.Priority
		_, _ = gradingagentrepo.InsertResult(ctx, d.Pool, gradingagentrepo.InsertResultInput{
			RunID: &msg.RunID, ConfigID: cfg.ID, SubmissionID: msg.SubmissionID,
			Status: gradingagentrepo.ItemFlagged, FlagReason: &reason,
			FlagPriority: &priority,
		})
		return gradingagentrepo.IncrementRunProgress(ctx, d.Pool, msg.RunID, false)
	}

	var rubricJSON []byte
	if len(preview.RubricScores) > 0 {
		rubricJSON, _ = json.Marshal(preview.RubricScores)
	}
	successIn := gradingAgentSuccessInput{
		Points: preview.Points, Comment: preview.Comment, Confidence: preview.Confidence,
		RubricJSON: rubricJSON, ModelID: preview.ModelID, PromptTokens: preview.PromptTokens,
		CompletionTokens: preview.CompletionTokens, CostUSD: preview.CostUSD, GradedByAI: preview.GradedByAI,
	}

	gateHold := &gradingAgentGateHoldInput{}
	if preview.Held != nil {
		gateHold.WouldHold = preview.Held.WouldHold
		gateHold.Reason = preview.Held.Reason
		gateHold.Queue = preview.Held.Queue
	}
	if hold, heldReason, queue := gradingAgentHoldDecision(cfg, preview.Confidence, gateHold); hold {
		return d.insertHeldGradingAgentResult(ctx, msg, cfg, successIn, heldReason, queue)
	}
	return d.finishGradingAgentSuccess(ctx, msg, cfg, assignRow, subRow, run, successIn)
}