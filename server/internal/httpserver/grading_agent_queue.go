package httpserver

import (
	"context"

	gradingagentrepo "github.com/lextures/lextures/server/internal/repos/gradingagent"
	"github.com/lextures/lextures/server/internal/repos/coursegrades"
	"github.com/lextures/lextures/server/internal/repos/coursemoduleassignments"
	"github.com/lextures/lextures/server/internal/repos/moduleassignmentsubmissions"
	"github.com/lextures/lextures/server/internal/gradingagentqueue"
	gradingagentsvc "github.com/lextures/lextures/server/internal/service/gradingagent"
)

// HandleGradingAgentQueueMessage grades one submission and writes a provisional grade.
func (d Deps) HandleGradingAgentQueueMessage(ctx context.Context, msg gradingagentqueue.QueueMessage) error {
	if d.Pool == nil {
		return nil
	}
	cfg, err := gradingagentrepo.GetConfigByItem(ctx, d.Pool, msg.ItemID)
	if err != nil || cfg == nil {
		return d.failGradingAgentItem(ctx, msg, "config not found", nil)
	}
	assignRow, err := coursemoduleassignments.GetForCourseItem(ctx, d.Pool, msg.CourseID, msg.ItemID)
	if err != nil || assignRow == nil {
		return d.failGradingAgentItem(ctx, msg, "assignment not found", nil)
	}
	subRow, err := moduleassignmentsubmissions.GetByIDForCourse(ctx, d.Pool, msg.CourseID, msg.SubmissionID)
	if err != nil || subRow == nil {
		return d.failGradingAgentItem(ctx, msg, "submission not found", nil)
	}
	run, err := gradingagentrepo.GetRun(ctx, d.Pool, msg.RunID)
	if err != nil || run == nil {
		return d.failGradingAgentItem(ctx, msg, "run not found", nil)
	}
	// Idempotency guard: if a result already exists for this (run, submission) pair the
	// message was already fully processed on a prior delivery. Ack without re-grading.
	if already, _ := gradingagentrepo.ResultExistsForRun(ctx, d.Pool, msg.RunID, msg.SubmissionID); already {
		return nil
	}
	runStatus, statusErr := d.gradingAgentRunStatus(ctx, msg.RunID)
	if statusErr != nil {
		return statusErr
	}
	if runStatus == gradingagentrepo.RunStatusCancelled {
		return d.skipGradingAgentItemForCancel(ctx, msg)
	}
	if over, budgetErr := d.gradingAgentRunOverBudget(ctx, run); budgetErr != nil {
		return budgetErr
	} else if over {
		return d.skipGradingAgentBudgetExceeded(ctx, msg, cfg.ID)
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
	content, err := svc.ResolveSubmissionContent(ctx, msg.CourseCode, subRow, gradingagentsvc.ResolveSubmissionContentOptions{
		TextEntryEnabled: d.graderAgentTextEntryGradingEnabled(),
		VisionEnabled:    d.graderAgentVisionGradingEnabled(),
	})
	if err != nil {
		return d.failGradingAgentItem(ctx, msg, err.Error(), nil)
	}
	if content.FailureReason != "" {
		modality := string(content.Modality)
		return d.failGradingAgentItem(ctx, msg, content.FailureReason, &modality)
	}
	useVision := content.Modality == gradingagentsvc.ModalityVision
	modelUser := cfg.CreatedBy
	if run.InitiatedBy != nil {
		modelUser = *run.InitiatedBy
	}

	var contentItemID string
	var rubricItemID string
	var workflowGraph *gradingagentsvc.WorkflowGraph
	var gradeSourceID string
	var compiled gradingagentsvc.CompiledWorkflow
	if wg, wgErr := gradingagentsvc.EffectiveWorkflowGraph(cfg.WorkflowGraph, cfg.Prompt, cfg.IncludeAssignmentContent, cfg.IncludeRubric); wgErr == nil && wg != nil {
		if c, compileErr := gradingagentsvc.CompileWorkflowGraph(wg, content.Text); compileErr == nil {
			compiled = c
			contentItemID = c.ContentItemID
			rubricItemID = c.RubricItemID
			workflowGraph = wg
			gradeSourceID = c.GradeSource
		}
	}
	contentRow, contentErr := d.assignmentRowForActivitySource(ctx, msg.CourseID, msg.ItemID, assignRow, contentItemID)
	if contentErr != nil {
		return d.failGradingAgentItem(ctx, msg, contentErr.Error(), nil)
	}
	rubricRow, rubricErr := d.assignmentRowForActivitySource(ctx, msg.CourseID, msg.ItemID, assignRow, rubricItemID)
	if rubricErr != nil {
		return d.failGradingAgentItem(ctx, msg, rubricErr.Error(), nil)
	}
	rubric, _ := gradingagentsvc.ParseAssignmentRubric(rubricRow)
	maxPoints := gradingagentsvc.MaxPointsFromAssignment(assignRow)

	if gradingAgentVisionComplexGraphBlocked(useVision, workflowGraph) {
		reason := gradingagentsvc.FailureVisionWorkflowUnsupported
		modality := string(gradingagentsvc.ModalityUnreadable)
		return d.failGradingAgentItem(ctx, msg, reason, &modality)
	}

	preview, execErr := d.executeGradingAgentPreview(ctx, svc, gradingAgentExecuteInput{
		ModelUser:                modelUser,
		Submissions:              content.Markdowns,
		SubmissionText:           content.Text,
		UseVision:                useVision,
		VisionImages:             content.ImageDataURLs,
		WorkflowGraph:            workflowGraph,
		GradeSourceID:            gradeSourceID,
		Compiled:                 compiled,
		ContentModality:          content.Modality,
		ContentMarkdown:          contentRow.Markdown,
		Rubric:                     rubric,
		MaxPoints:                  maxPoints,
		CourseCode:               msg.CourseCode,
		SubmissionID:             msg.SubmissionID,
		InstructorPrompt:         cfg.Prompt,
		IncludeAssignmentContent: cfg.IncludeAssignmentContent,
		IncludeRubric:            cfg.IncludeRubric,
		ConfigModelID:            cfg.ModelID,
	})
	if execErr != nil {
		return d.failGradingAgentItem(ctx, msg, execErr.Error(), nil)
	}
	if err := d.persistGradingAgentPreview(ctx, msg, cfg, assignRow, subRow, run, preview); err != nil {
		return d.failGradingAgentItem(ctx, msg, "failed to write grade", nil)
	}
	d.recordGradingAgentQueueUsage(ctx, modelUser, msg.CourseCode, msg.CourseID, preview, true)
	return nil
}

func (d Deps) failGradingAgentItem(ctx context.Context, msg gradingagentqueue.QueueMessage, reason string, modality *string) error {
	_, _ = gradingagentrepo.InsertResult(ctx, d.Pool, gradingagentrepo.InsertResultInput{
		RunID: &msg.RunID, ConfigID: msg.ConfigID, SubmissionID: msg.SubmissionID,
		Status: gradingagentrepo.ItemFailed, Error: &reason, InputModality: modality,
	})
	_ = gradingagentrepo.IncrementRunProgress(ctx, d.Pool, msg.RunID, true)
	return nil
}

func gradingAgentStrPtr(s string) *string { return &s }
