package httpserver

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/google/uuid"

	gradingagentrepo "github.com/lextures/lextures/server/internal/repos/gradingagent"
	"github.com/lextures/lextures/server/internal/repos/coursegrades"
	"github.com/lextures/lextures/server/internal/repos/coursemoduleassignments"
	"github.com/lextures/lextures/server/internal/repos/moduleassignmentsubmissions"
	"github.com/lextures/lextures/server/internal/gradingagentqueue"
	"github.com/lextures/lextures/server/internal/service/codeexecution"
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
	submissions := content.Markdowns
	submissionText := content.Text
	useVision := content.Modality == gradingagentsvc.ModalityVision
	visionImages := content.ImageDataURLs
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
		if c, compileErr := gradingagentsvc.CompileWorkflowGraph(wg, submissionText); compileErr == nil {
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

	if useVision && workflowGraph != nil && gradingagentsvc.WorkflowRequiresGraphExecution(workflowGraph) {
		reason := gradingagentsvc.FailureVisionWorkflowUnsupported
		modality := string(gradingagentsvc.ModalityUnreadable)
		return d.failGradingAgentItem(ctx, msg, reason, &modality)
	}
	if workflowGraph != nil && gradingagentsvc.WorkflowRequiresGraphExecution(workflowGraph) {
		dryRunModelID, modelErr := d.resolveGraderAgentModelID(ctx, modelUser, compiled.ScoreRequest.ModelID, cfg.ModelID)
		if modelErr != nil {
			return d.failGradingAgentItem(ctx, msg, modelErr.Error(), nil)
		}
		if gradingagentsvc.WorkflowUsesLLM(workflowGraph) {
			dec, _ := aigateway.Evaluate(ctx, d.Pool, d.aiGatewayConfig(), modelUser, nil, aigateway.FeatureGraderAgent, dryRunModelID, aigateway.ContentHash(gradingagentsvc.ContentHashInput(compiled.ScoreRequest.InstructorPrompt, submissionText)))
			if !dec.Allowed {
				return d.failGradingAgentItem(ctx, msg, "AI processing blocked", nil)
			}
			if svc.Client == nil {
				return d.failGradingAgentItem(ctx, msg, "AI provider not configured", nil)
			}
		}
		preview, execErr := gradingagentsvc.ExecuteWorkflowDryRun(ctx, gradingagentsvc.DryRunExecutionInput{
			Graph:           workflowGraph,
			Submissions:     submissions,
			InputModality:   content.Modality,
			SubmissionID:    msg.SubmissionID,
			CourseCode:      msg.CourseCode,
			DefaultMarkdown: contentRow.Markdown,
			DefaultRubric:   rubric,
			MaxPoints:       maxPoints,
			ModelID:         dryRunModelID,
			LoadOriginalityReports: d.loadOriginalityReportsForGraderAgent,
			LoadReferenceFile: func(ctx context.Context, courseCode string, fileID uuid.UUID) (string, error) {
				return svc.LoadReferenceFileMarkdown(ctx, courseCode, fileID)
			},
			Runner:     svc,
			CodeRunner: codeexecution.New(),
		})
		if execErr != nil {
			return d.failGradingAgentItem(ctx, msg, execErr.Error(), nil)
		}
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
		comment := preview.Comment
		conf := preview.Confidence
		pts := preview.SuggestedPoints
		gateHold := &gradingAgentGateHoldInput{}
		if preview.Held != nil {
			gateHold.WouldHold = preview.Held.WouldHold
			gateHold.Reason = preview.Held.Reason
			gateHold.Queue = preview.Held.Queue
		}
		if hold, heldReason, queue := gradingAgentHoldDecision(cfg, conf, gateHold); hold {
			if err := d.insertHeldGradingAgentResult(ctx, msg, cfg, gradingAgentSuccessInput{
				Points: pts, Comment: comment, Confidence: conf,
			}, heldReason, queue); err != nil {
				return d.failGradingAgentItem(ctx, msg, "failed to record held grade", nil)
			}
			return nil
		}
		var rubricJSON []byte
		if len(preview.RubricScores) > 0 {
			rubricJSON, _ = json.Marshal(preview.RubricScores)
		}
		if err := d.finishGradingAgentSuccess(ctx, msg, cfg, assignRow, subRow, run, gradingAgentSuccessInput{
			Points: pts, Comment: comment, Confidence: conf, RubricJSON: rubricJSON, GradedByAI: true,
		}); err != nil {
			return d.failGradingAgentItem(ctx, msg, "failed to write grade", nil)
		}
		return nil
	}

	if workflowGraph != nil && gradeSourceID != "" {
		var gradeSource *gradingagentsvc.WorkflowNode
		for i := range workflowGraph.Nodes {
			if workflowGraph.Nodes[i].ID == gradeSourceID {
				gradeSource = &workflowGraph.Nodes[i]
				break
			}
		}
		if gradeSource != nil {
			switch gradeSource.Type {
			case gradingagentsvc.NodeTypeCodeTestRunner, gradingagentsvc.NodeTypeCriterionGrader, gradingagentsvc.NodeTypeAI:
				dryRunModelID := compiled.ScoreRequest.ModelID
				if gradeSource.Type != gradingagentsvc.NodeTypeCodeTestRunner {
					var modelErr error
					dryRunModelID, modelErr = d.resolveGraderAgentModelID(ctx, modelUser, dryRunModelID, cfg.ModelID)
					if modelErr != nil {
						return d.failGradingAgentItem(ctx, msg, modelErr.Error(), nil)
					}
					dec, _ := aigateway.Evaluate(ctx, d.Pool, d.aiGatewayConfig(), modelUser, nil, aigateway.FeatureGraderAgent, dryRunModelID, aigateway.ContentHash(gradingagentsvc.ContentHashInput(compiled.ScoreRequest.InstructorPrompt, submissionText)))
					if !dec.Allowed {
						return d.failGradingAgentItem(ctx, msg, "AI processing blocked", nil)
					}
					if svc.Client == nil {
						return d.failGradingAgentItem(ctx, msg, "AI provider not configured", nil)
					}
				}
				preview, execErr := gradingagentsvc.ExecuteWorkflowDryRun(ctx, gradingagentsvc.DryRunExecutionInput{
					Graph:           workflowGraph,
					Submissions:     submissions,
					InputModality:   content.Modality,
					SubmissionID:    msg.SubmissionID,
					CourseCode:      msg.CourseCode,
					DefaultMarkdown: contentRow.Markdown,
					DefaultRubric:   rubric,
					MaxPoints:       maxPoints,
					ModelID:         dryRunModelID,
					LoadOriginalityReports: d.loadOriginalityReportsForGraderAgent,
					LoadReferenceFile: func(ctx context.Context, courseCode string, fileID uuid.UUID) (string, error) {
						return svc.LoadReferenceFileMarkdown(ctx, courseCode, fileID)
					},
					Runner:     svc,
					CodeRunner: codeexecution.New(),
				})
				if execErr != nil {
					return d.failGradingAgentItem(ctx, msg, execErr.Error(), nil)
				}
				comment := preview.Comment
				conf := preview.Confidence
				pts := preview.SuggestedPoints
				var rubricJSON []byte
				if len(preview.RubricScores) > 0 {
					rubricJSON, _ = json.Marshal(preview.RubricScores)
				}
				gradedByAI := gradeSource.Type != gradingagentsvc.NodeTypeCodeTestRunner
				if err := d.finishGradingAgentSuccess(ctx, msg, cfg, assignRow, subRow, run, gradingAgentSuccessInput{
					Points: pts, Comment: comment, Confidence: conf, RubricJSON: rubricJSON, GradedByAI: gradedByAI,
				}); err != nil {
					return d.failGradingAgentItem(ctx, msg, "failed to write grade", nil)
				}
				return nil
			}
		}
	}

	modelID, modelErr := d.resolveGraderAgentModelID(ctx, modelUser, "", cfg.ModelID)
	if modelErr != nil {
		return d.failGradingAgentItem(ctx, msg, modelErr.Error(), nil)
	}
	dec, _ := aigateway.Evaluate(ctx, d.Pool, d.aiGatewayConfig(), modelUser, nil, aigateway.FeatureGraderAgent, modelID, aigateway.ContentHash(gradingagentsvc.ContentHashInput(cfg.Prompt, submissionText)))
	if !dec.Allowed {
		return d.failGradingAgentItem(ctx, msg, "AI processing blocked", nil)
	}
	if svc.Client == nil {
		return d.failGradingAgentItem(ctx, msg, "AI provider not configured", nil)
	}
	scoreReq := gradingagentsvc.ScoreRequest{
		InstructorPrompt:         cfg.Prompt,
		IncludeAssignmentContent: cfg.IncludeAssignmentContent,
		IncludeRubric:            cfg.IncludeRubric,
		ModelID:                  modelID,
		SubmissionText:           submissionText,
	}
	if compiled.GradeSource != "" {
		scoreReq = compiled.ScoreRequest
		scoreReq.ModelID = modelID
	}
	scoreReq.AssignmentMarkdown = contentRow.Markdown
	scoreReq.Rubric = rubric
	scoreReq.MaxPoints = maxPoints
	if workflowGraph != nil && strings.TrimSpace(gradeSourceID) != "" {
		scoreReq.InstructorPrompt = gradingagentsvc.SubstituteWorkflowPromptVariables(
			workflowGraph,
			gradeSourceID,
			scoreReq.InstructorPrompt,
			gradingagentsvc.PromptVariableContext{
				Submissions:     submissions,
				ContentMarkdown: contentRow.Markdown,
				Rubric:          rubric,
			},
		)
	}
	var result gradingagentsvc.ScoreResult
	if useVision {
		result, err = svc.ScoreWithVision(ctx, scoreReq, visionImages)
	} else {
		result, err = svc.Score(ctx, scoreReq)
	}
	if err != nil {
		return d.failGradingAgentItem(ctx, msg, err.Error(), nil)
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
	if err := d.finishGradingAgentSuccess(ctx, msg, cfg, assignRow, subRow, run, gradingAgentSuccessInput{
		Points: pts, Comment: comment, Confidence: conf, RubricJSON: rubricJSON,
		ModelID: &model, PromptTokens: &pt, CompletionTokens: &ct, CostUSD: &cost, GradedByAI: true,
	}); err != nil {
		return d.failGradingAgentItem(ctx, msg, "failed to write grade", nil)
	}
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