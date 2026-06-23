package httpserver

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/coder/websocket"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/lextures/lextures/server/internal/models/assignmentrubric"
	"github.com/lextures/lextures/server/internal/repos/enrollment"
	gradingagentrepo "github.com/lextures/lextures/server/internal/repos/gradingagent"
	"github.com/lextures/lextures/server/internal/repos/moduleassignmentsubmissions"
	"github.com/lextures/lextures/server/internal/repos/rbac"
	"github.com/lextures/lextures/server/internal/service/codeexecution"
	gradingagentsvc "github.com/lextures/lextures/server/internal/service/gradingagent"
	aigateway "github.com/lextures/lextures/server/internal/service/aigateway"
	"github.com/lextures/lextures/server/internal/service/openrouter"
)

type graderAgentDryRunWSFirstMessage struct {
	AuthToken     string                          `json:"authToken"`
	SubmissionID  string                          `json:"submissionId"`
	WorkflowGraph *gradingagentsvc.WorkflowGraph `json:"workflowGraph"`
}

// handleGraderAgentDryRunWS streams step-by-step workflow dry-run progress.
// GET /api/v1/courses/{course_code}/assignments/{item_id}/grader-agent/dry-run/ws
func (d Deps) handleGraderAgentDryRunWS() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		if !d.graderAgentEnabled() || d.JWTSigner == nil || d.Pool == nil {
			http.Error(w, "server misconfiguration", http.StatusServiceUnavailable)
			return
		}

		courseCode := chi.URLParam(r, "course_code")
		itemID, err := uuid.Parse(chi.URLParam(r, "item_id"))
		if err != nil || courseCode == "" {
			http.Error(w, "invalid request", http.StatusBadRequest)
			return
		}

		conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{OriginPatterns: []string{"*"}})
		if err != nil {
			return
		}
		defer func() { _ = conn.Close(websocket.StatusNormalClosure, "") }()

		readCtx, cancelRead := context.WithTimeout(r.Context(), 2*time.Minute)
		defer cancelRead()
		typ, payload, err := conn.Read(readCtx)
		if err != nil || typ != websocket.MessageText {
			return
		}

		var first graderAgentDryRunWSFirstMessage
		if err := json.Unmarshal(payload, &first); err != nil || strings.TrimSpace(first.AuthToken) == "" {
			_ = wsWriteJSON(r.Context(), conn, gradingagentsvc.DryRunEvent{Type: "error", Message: "Invalid first message. Send authToken, submissionId, and workflowGraph."})
			return
		}
		u, err := d.JWTSigner.Verify(r.Context(), first.AuthToken)
		if err != nil {
			return
		}
		viewer, err := uuid.Parse(u.UserID)
		if err != nil {
			return
		}
		if !d.userCanAccessGraderAgent(r.Context(), courseCode, viewer) {
			return
		}

		submissionID, err := uuid.Parse(strings.TrimSpace(first.SubmissionID))
		if err != nil {
			_ = wsWriteJSON(r.Context(), conn, gradingagentsvc.DryRunEvent{Type: "error", Message: "Invalid submission id."})
			return
		}
		if first.WorkflowGraph == nil {
			_ = wsWriteJSON(r.Context(), conn, gradingagentsvc.DryRunEvent{Type: "error", Message: "Workflow graph is required."})
			return
		}

		cid, assignRow, loadErr := d.loadAssignmentForSubmissionsByIDs(r.Context(), courseCode, itemID)
		if loadErr != nil || assignRow == nil || cid == nil {
			_ = wsWriteJSON(r.Context(), conn, gradingagentsvc.DryRunEvent{Type: "error", Message: "Assignment not found."})
			return
		}
		subRow, err := moduleassignmentsubmissions.GetByIDForCourse(r.Context(), d.Pool, *cid, submissionID)
		if err != nil || subRow == nil {
			_ = wsWriteJSON(r.Context(), conn, gradingagentsvc.DryRunEvent{Type: "error", Message: "Submission not found."})
			return
		}
		if subRow.AttachmentFileID == nil {
			_ = wsWriteJSON(r.Context(), conn, gradingagentsvc.DryRunEvent{Type: "error", Message: "This submission has no readable text attachment."})
			return
		}

		svc := d.gradingAgentService()
		submissions, err := svc.LoadSubmissionMarkdownsForSubmission(r.Context(), courseCode, subRow)
		if err != nil {
			_ = wsWriteJSON(r.Context(), conn, gradingagentsvc.DryRunEvent{Type: "error", Message: dryRunSubmissionLoadMessage(err)})
			return
		}
		submissionText := gradingagentsvc.JoinSubmissions(submissions)

		compiled, compileErr := gradingagentsvc.CompileWorkflowGraph(first.WorkflowGraph, submissionText)
		if compileErr != nil {
			msg := compileErr.Error()
			if ve, ok := compileErr.(gradingagentsvc.ValidationError); ok {
				msg = ve.Message
			}
			_ = wsWriteJSON(r.Context(), conn, gradingagentsvc.DryRunEvent{Type: "error", Message: msg})
			return
		}

		needsLLM := gradingagentsvc.WorkflowUsesLLM(first.WorkflowGraph)
		var modelID string
		var governancePrompt string
		if needsLLM {
			explicitModel := compiled.ScoreRequest.ModelID
			var modelErr error
			modelID, modelErr = d.resolveGraderAgentModelID(r.Context(), viewer, explicitModel, nil)
			if modelErr != nil {
				_ = wsWriteJSON(r.Context(), conn, gradingagentsvc.DryRunEvent{Type: "error", Message: gradingagentsvc.UserFacingScoreError(modelErr)})
				return
			}
			governancePrompt = compiled.ScoreRequest.InstructorPrompt
			if blockMsg, blocked := d.evaluateAIGatewayBlock(r.Context(), viewer, aigateway.FeatureGraderAgent, modelID, gradingagentsvc.ContentHashInput(governancePrompt, submissionText)); blocked {
				_ = wsWriteJSON(r.Context(), conn, gradingagentsvc.DryRunEvent{Type: "error", Message: blockMsg})
				return
			}
			if d.openRouterClient() == nil || strings.TrimSpace(d.effectiveConfig().OpenRouterAPIKey) == "" {
				_ = wsWriteJSON(r.Context(), conn, gradingagentsvc.DryRunEvent{Type: "error", Message: "AI provider is not configured."})
				return
			}
		}

		contentRow, contentErr := d.assignmentRowForActivitySource(r.Context(), *cid, itemID, assignRow, compiled.ContentItemID)
		if contentErr != nil {
			_ = wsWriteJSON(r.Context(), conn, gradingagentsvc.DryRunEvent{Type: "error", Message: contentErr.Error()})
			return
		}
		rubricRow, rubricErr := d.assignmentRowForActivitySource(r.Context(), *cid, itemID, assignRow, compiled.RubricItemID)
		if rubricErr != nil {
			_ = wsWriteJSON(r.Context(), conn, gradingagentsvc.DryRunEvent{Type: "error", Message: rubricErr.Error()})
			return
		}
		rubric, _ := gradingagentsvc.ParseAssignmentRubric(rubricRow)
		maxPoints := gradingagentsvc.MaxPointsFromAssignment(assignRow)

		runCtx := r.Context()
		emit := func(ev gradingagentsvc.DryRunEvent) {
			_ = wsWriteJSON(runCtx, conn, ev)
		}
		emit(gradingagentsvc.DryRunEvent{Type: "log", Level: "info", Message: "Starting dry run…"})

		preview, execErr := gradingagentsvc.ExecuteWorkflowDryRun(runCtx, gradingagentsvc.DryRunExecutionInput{
			Graph:           first.WorkflowGraph,
			Submissions:     submissions,
			DefaultMarkdown: contentRow.Markdown,
			DefaultRubric:   rubric,
			MaxPoints:       maxPoints,
			ModelID:         modelID,
			ResolveActivity: func(overrideItemID string) (string, *assignmentrubric.RubricDefinition, error) {
				row, rowErr := d.assignmentRowForActivitySource(runCtx, *cid, itemID, assignRow, overrideItemID)
				if rowErr != nil {
					return "", nil, rowErr
				}
				r, _ := gradingagentsvc.ParseAssignmentRubric(row)
				return row.Markdown, r, nil
			},
			Runner:     svc,
			CodeRunner: codeexecution.New(),
			Emit:       emit,
		})
		if execErr != nil {
			msg := gradingagentsvc.UserFacingScoreError(execErr)
			if ve, ok := execErr.(gradingagentsvc.ValidationError); ok {
				msg = ve.Message
			}
			emit(gradingagentsvc.DryRunEvent{Type: "error", Message: msg})
			return
		}

		if needsLLM {
			d.recordAIUsage(runCtx, AIUsageMeta{
				UserID: viewer, CourseCode: courseCode, Feature: aigateway.FeatureGraderAgent, Model: modelID,
			}, openrouter.UsageInfo{
				PromptTokens:     preview.PromptTokens,
				CompletionTokens: preview.CompletionTokens,
				TotalTokens:      preview.PromptTokens + preview.CompletionTokens,
			}, true)
			d.logAIInferenceAllowed(r, viewer, aigateway.FeatureGraderAgent, modelID, gradingagentsvc.ContentHashInput(governancePrompt, submissionText), aigateway.Decision{Allowed: true, OptInConfirmed: true})
		}

		cfg, _ := gradingagentrepo.GetConfigByItem(runCtx, d.Pool, itemID)
		var configID uuid.UUID
		if cfg != nil {
			configID = cfg.ID
		} else {
			saved, saveErr := gradingagentrepo.UpsertConfig(runCtx, d.Pool, gradingagentrepo.UpsertConfigInput{
				CourseID: *cid, ModuleItemID: itemID, Status: gradingagentrepo.StatusDraft,
				Prompt: governancePrompt, IncludeAssignmentContent: compiled.ScoreRequest.IncludeAssignmentContent,
				IncludeRubric: compiled.ScoreRequest.IncludeRubric, CreatedBy: viewer,
			})
			if saveErr == nil && saved != nil {
				configID = saved.ID
			}
		}
		if configID != uuid.Nil {
			comment := preview.Comment
			conf := preview.Confidence
			pt := preview.PromptTokens
			ct := preview.CompletionTokens
			pts := preview.SuggestedPoints
			_, _ = gradingagentrepo.InsertResult(runCtx, d.Pool, gradingagentrepo.InsertResultInput{
				ConfigID: configID, SubmissionID: submissionID, IsDryRun: true,
				SuggestedPoints: &pts, Comment: &comment, Confidence: &conf,
				Status: gradingagentrepo.ItemSuggested, ModelID: &modelID,
				PromptTokens: &pt, CompletionTokens: &ct,
			})
		}

		emit(gradingagentsvc.DryRunEvent{Type: "complete"})
	}
}

func (d Deps) userCanAccessGraderAgent(ctx context.Context, courseCode string, viewer uuid.UUID) bool {
	if !d.graderAgentEnabled() || d.Pool == nil {
		return false
	}
	hasAccess, err := enrollment.UserHasAccess(ctx, d.Pool, courseCode, viewer)
	if err != nil || !hasAccess {
		return false
	}
	has, err := rbac.UserHasPermission(ctx, d.Pool, viewer, "course:"+courseCode+":item:create")
	return err == nil && has
}

func dryRunSubmissionLoadMessage(err error) string {
	msg := "Could not read submission content."
	switch {
	case strings.Contains(err.Error(), "empty submission text"):
		msg = "Submission file is empty."
	case strings.Contains(err.Error(), "no submission text"):
		msg = "This submission has no readable text attachment."
	case strings.Contains(err.Error(), "submission file not found"):
		msg = "Submission attachment record is missing."
	}
	return msg
}