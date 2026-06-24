package httpserver

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/lextures/lextures/server/internal/models/gradecomment"
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
	submissions, err := svc.LoadSubmissionMarkdownsForSubmission(ctx, msg.CourseCode, subRow)
	if err != nil {
		return d.failGradingAgentItem(ctx, msg, err.Error())
	}
	submissionText := gradingagentsvc.JoinSubmissions(submissions)
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
		return d.failGradingAgentItem(ctx, msg, contentErr.Error())
	}
	rubricRow, rubricErr := d.assignmentRowForActivitySource(ctx, msg.CourseID, msg.ItemID, assignRow, rubricItemID)
	if rubricErr != nil {
		return d.failGradingAgentItem(ctx, msg, rubricErr.Error())
	}
	rubric, _ := gradingagentsvc.ParseAssignmentRubric(rubricRow)
	maxPoints := gradingagentsvc.MaxPointsFromAssignment(assignRow)

	if workflowGraph != nil && gradingagentsvc.WorkflowRequiresGraphExecution(workflowGraph) {
		dryRunModelID, modelErr := d.resolveGraderAgentModelID(ctx, modelUser, compiled.ScoreRequest.ModelID, cfg.ModelID)
		if modelErr != nil {
			return d.failGradingAgentItem(ctx, msg, modelErr.Error())
		}
		if gradingagentsvc.WorkflowUsesLLM(workflowGraph) {
			dec, _ := aigateway.Evaluate(ctx, d.Pool, d.aiGatewayConfig(), modelUser, nil, aigateway.FeatureGraderAgent, dryRunModelID, aigateway.ContentHash(gradingagentsvc.ContentHashInput(compiled.ScoreRequest.InstructorPrompt, submissionText)))
			if !dec.Allowed {
				return d.failGradingAgentItem(ctx, msg, "AI processing blocked")
			}
			if svc.Client == nil {
				return d.failGradingAgentItem(ctx, msg, "AI provider not configured")
			}
		}
		preview, execErr := gradingagentsvc.ExecuteWorkflowDryRun(ctx, gradingagentsvc.DryRunExecutionInput{
			Graph:           workflowGraph,
			Submissions:     submissions,
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
			return d.failGradingAgentItem(ctx, msg, execErr.Error())
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
		if preview.Held != nil && preview.Held.WouldHold {
			heldReason := preview.Held.Reason
			queue := preview.Held.Queue
			now := time.Now()
			_, _ = gradingagentrepo.InsertResult(ctx, d.Pool, gradingagentrepo.InsertResultInput{
				RunID: &msg.RunID, ConfigID: cfg.ID, SubmissionID: msg.SubmissionID,
				SuggestedPoints: &pts, Comment: &comment, Confidence: &conf,
				Status: gradingagentrepo.ItemSuggested, HeldReason: &heldReason,
				HeldAt: &now, HeldQueue: &queue,
			})
			return gradingagentrepo.IncrementRunProgress(ctx, d.Pool, msg.RunID, false)
		}
		var rubricJSON []byte
		if len(preview.RubricScores) > 0 {
			rubricJSON, _ = json.Marshal(preview.RubricScores)
		}
		posting := "manual"
		if strings.TrimSpace(assignRow.PostingPolicy) == "automatic" && cfg.PostPolicy == "auto_post" {
			posting = "automatic"
		}
		_, commentJSON, flatComment, _ := gradecomment.Append(nil, gradecomment.Comment{
			DisplayName: "Grading agent",
			Body:        comment,
			Source:      "lextures",
		})
		if err := coursegrades.UpsertCellWithFlags(ctx, d.Pool, msg.CourseID, subRow.SubmittedBy, msg.ItemID, pts, rubricJSON, flatComment, commentJSON, posting, true); err != nil {
			return d.failGradingAgentItem(ctx, msg, "failed to write grade")
		}
		_, _ = gradingagentrepo.InsertResult(ctx, d.Pool, gradingagentrepo.InsertResultInput{
			RunID: &msg.RunID, ConfigID: cfg.ID, SubmissionID: msg.SubmissionID,
			SuggestedPoints: &pts, Comment: &comment, Confidence: &conf,
			Status: gradingagentrepo.ItemApplied,
		})
		return gradingagentrepo.IncrementRunProgress(ctx, d.Pool, msg.RunID, false)
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
						return d.failGradingAgentItem(ctx, msg, modelErr.Error())
					}
					dec, _ := aigateway.Evaluate(ctx, d.Pool, d.aiGatewayConfig(), modelUser, nil, aigateway.FeatureGraderAgent, dryRunModelID, aigateway.ContentHash(gradingagentsvc.ContentHashInput(compiled.ScoreRequest.InstructorPrompt, submissionText)))
					if !dec.Allowed {
						return d.failGradingAgentItem(ctx, msg, "AI processing blocked")
					}
					if svc.Client == nil {
						return d.failGradingAgentItem(ctx, msg, "AI provider not configured")
					}
				}
				preview, execErr := gradingagentsvc.ExecuteWorkflowDryRun(ctx, gradingagentsvc.DryRunExecutionInput{
					Graph:           workflowGraph,
					Submissions:     submissions,
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
					return d.failGradingAgentItem(ctx, msg, execErr.Error())
				}
				comment := preview.Comment
				conf := preview.Confidence
				pts := preview.SuggestedPoints
				var rubricJSON []byte
				if len(preview.RubricScores) > 0 {
					rubricJSON, _ = json.Marshal(preview.RubricScores)
				}
				posting := "manual"
				if strings.TrimSpace(assignRow.PostingPolicy) == "automatic" && cfg.PostPolicy == "auto_post" {
					posting = "automatic"
				}
				gradedByAI := gradeSource.Type != gradingagentsvc.NodeTypeCodeTestRunner
				_, commentJSON, flatComment, _ := gradecomment.Append(nil, gradecomment.Comment{
					DisplayName: "Grading agent",
					Body:        comment,
					Source:      "lextures",
				})
				if err := coursegrades.UpsertCellWithFlags(ctx, d.Pool, msg.CourseID, subRow.SubmittedBy, msg.ItemID, pts, rubricJSON, flatComment, commentJSON, posting, gradedByAI); err != nil {
					return d.failGradingAgentItem(ctx, msg, "failed to write grade")
				}
				_, _ = gradingagentrepo.InsertResult(ctx, d.Pool, gradingagentrepo.InsertResultInput{
					RunID: &msg.RunID, ConfigID: cfg.ID, SubmissionID: msg.SubmissionID,
					SuggestedPoints: &pts, Comment: &comment, Confidence: &conf,
					Status: gradingagentrepo.ItemApplied,
				})
				return gradingagentrepo.IncrementRunProgress(ctx, d.Pool, msg.RunID, false)
			}
		}
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
	result, err := svc.Score(ctx, scoreReq)
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
	_, commentJSON, flatComment, _ := gradecomment.Append(nil, gradecomment.Comment{
		DisplayName: "Grading agent",
		Body:        comment,
		Source:      "lextures",
	})
	if err := coursegrades.UpsertCellWithFlags(ctx, d.Pool, msg.CourseID, subRow.SubmittedBy, msg.ItemID, pts, rubricJSON, flatComment, commentJSON, posting, gradedByAI); err != nil {
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