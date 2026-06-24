package httpserver

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/lextures/lextures/server/internal/apierr"
	"github.com/lextures/lextures/server/internal/gradingagentqueue"
	"github.com/lextures/lextures/server/internal/repos/course"
	"github.com/lextures/lextures/server/internal/repos/coursemoduleassignments"
	gradingagentrepo "github.com/lextures/lextures/server/internal/repos/gradingagent"
	"github.com/lextures/lextures/server/internal/repos/coursegrades"
	"github.com/lextures/lextures/server/internal/repos/moduleassignmentsubmissions"
	"github.com/lextures/lextures/server/internal/repos/originalityreports"
	"github.com/lextures/lextures/server/internal/repos/notificationsinbox"
	"github.com/lextures/lextures/server/internal/repos/rbac"
	"github.com/lextures/lextures/server/internal/repos/user"
	gradingagentsvc "github.com/lextures/lextures/server/internal/service/gradingagent"
	aigateway "github.com/lextures/lextures/server/internal/service/aigateway"
	"github.com/lextures/lextures/server/internal/service/openrouter"
)

func (d Deps) graderAgentEnabled() bool {
	return d.effectiveConfig().GraderAgentEnabled
}

func (d Deps) requireGraderAgentAccess(w http.ResponseWriter, r *http.Request) (courseCode string, viewer uuid.UUID, ok bool) {
	if !d.graderAgentEnabled() {
		apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Grader agent is not enabled.")
		return "", uuid.Nil, false
	}
	courseCode, viewer, ok = d.requireCourseAccess(w, r)
	if !ok {
		return "", uuid.Nil, false
	}
	has, err := rbac.UserHasPermission(r.Context(), d.Pool, viewer, "course:"+courseCode+":item:create")
	if err != nil {
		apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to verify permissions.")
		return "", uuid.Nil, false
	}
	if !has {
		apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "You do not have permission to grade.")
		return "", uuid.Nil, false
	}
	return courseCode, viewer, true
}

func (d Deps) loadOriginalityReportsForGraderAgent(ctx context.Context, submissionID uuid.UUID) ([]gradingagentsvc.OriginalityReportRow, error) {
	if d.Pool == nil {
		return nil, nil
	}
	reports, err := originalityreports.ListBySubmission(ctx, d.Pool, submissionID)
	if err != nil {
		return nil, err
	}
	out := make([]gradingagentsvc.OriginalityReportRow, 0, len(reports))
	for _, r := range reports {
		out = append(out, gradingagentsvc.OriginalityReportRow{
			Provider:      r.Provider,
			Status:        r.Status,
			SimilarityPct: r.SimilarityPct,
			AIProbability: r.AIProbability,
			ReportURL:     r.ReportURL,
			UpdatedAt:     r.UpdatedAt,
		})
	}
	return out, nil
}

func (d Deps) gradingAgentService() *gradingagentsvc.Service {
	cfg := d.effectiveConfig()
	return &gradingagentsvc.Service{
		Client:    d.openRouterClient(),
		Storage:   d.Storage,
		FilesRoot: cfg.CourseFilesRoot,
		Pool:      d.Pool,
	}
}

// resolveGraderAgentModelID picks the OpenRouter model: explicit override, saved config, then Intelligence → Models.
func (d Deps) resolveGraderAgentModelID(ctx context.Context, userID uuid.UUID, explicit string, configModel *string) (string, error) {
	if s := strings.TrimSpace(explicit); s != "" {
		return s, nil
	}
	if configModel != nil {
		if s := strings.TrimSpace(*configModel); s != "" {
			return s, nil
		}
	}
	if d.Pool == nil {
		return "", fmt.Errorf("grader agent model not configured")
	}
	model, err := user.GetGraderAgentModelID(ctx, d.Pool, userID)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(model) == "" {
		return "", fmt.Errorf("grader agent model not configured")
	}
	return model, nil
}

func graderAgentConfigToJSON(row *gradingagentrepo.ConfigRow) map[string]any {
	if row == nil {
		return nil
	}
	out := map[string]any{
		"id":                       row.ID.String(),
		"prompt":                   row.Prompt,
		"includeAssignmentContent": row.IncludeAssignmentContent,
		"includeRubric":            row.IncludeRubric,
		"status":                   string(row.Status),
		"autoGradeNew":             row.AutoGradeNew,
		"updatedAt":                row.UpdatedAt.UTC().Format("2006-01-02T15:04:05.000000Z"),
	}
	if row.ModelID != nil {
		out["modelId"] = *row.ModelID
	}
	if g, err := gradingagentsvc.EffectiveWorkflowGraph(row.WorkflowGraph, row.Prompt, row.IncludeAssignmentContent, row.IncludeRubric); err == nil && g != nil {
		out["workflowGraph"] = g
	}
	return out
}

func (d Deps) assignmentRowForActivitySource(
	ctx context.Context,
	courseID uuid.UUID,
	defaultItemID uuid.UUID,
	defaultRow *coursemoduleassignments.CourseItemAssignmentRow,
	overrideItemID string,
) (*coursemoduleassignments.CourseItemAssignmentRow, error) {
	overrideItemID = strings.TrimSpace(overrideItemID)
	if overrideItemID == "" || overrideItemID == defaultItemID.String() {
		return defaultRow, nil
	}
	itemID, err := uuid.Parse(overrideItemID)
	if err != nil {
		return nil, fmt.Errorf("invalid activity assignment id")
	}
	row, err := coursemoduleassignments.GetForCourseItem(ctx, d.Pool, courseID, itemID)
	if err != nil || row == nil {
		return nil, fmt.Errorf("activity assignment not found")
	}
	return row, nil
}

func writeGraderAgentValidationError(w http.ResponseWriter, err error) {
	if ve, ok := err.(gradingagentsvc.ValidationError); ok {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]string{
				"code":    apierr.CodeInvalidInput,
				"message": ve.Message,
			},
			"field": ve.Field,
		})
		return
	}
	apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, err.Error())
}

func (d Deps) handleListCourseGradingAgents() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		courseCode, _, ok := d.requireGraderAgentAccess(w, r)
		if !ok {
			return
		}
		cid, found, err := d.courseIDFromCode(r.Context(), courseCode)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load course.")
			return
		}
		if !found {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Course not found.")
			return
		}
		rows, err := gradingagentrepo.ListConfigsByCourse(r.Context(), d.Pool, cid)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load grading agents.")
			return
		}
		agents := make([]map[string]any, 0, len(rows))
		for _, row := range rows {
			agents = append(agents, map[string]any{
				"id":                 row.ID.String(),
				"itemId":             row.ModuleItemID.String(),
				"assignmentTitle":    row.AssignmentTitle,
				"assignmentArchived": row.AssignmentArchived,
				"status":             string(row.Status),
				"autoGradeNew":       row.AutoGradeNew,
				"hasWorkflowGraph":   row.HasWorkflowGraph,
				"updatedAt":          row.UpdatedAt.UTC().Format("2006-01-02T15:04:05.000000Z"),
			})
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{"agents": agents})
	}
}

func (d Deps) handleGetGraderAgentConfig() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		courseCode, _, ok := d.requireGraderAgentAccess(w, r)
		if !ok {
			return
		}
		itemID, err := uuid.Parse(chi.URLParam(r, "item_id"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid assignment id.")
			return
		}
		_, assignRow, ok := d.loadAssignmentForSubmissions(w, r, courseCode, itemID)
		if !ok || assignRow == nil {
			return
		}
		cfg, err := gradingagentrepo.GetConfigByItem(r.Context(), d.Pool, itemID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load grader agent.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		if cfg == nil {
			_ = json.NewEncoder(w).Encode(map[string]any{"config": nil})
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"config": graderAgentConfigToJSON(cfg)})
	}
}

type putGraderAgentConfigBody struct {
	Prompt                   string                           `json:"prompt"`
	IncludeAssignmentContent bool                             `json:"includeAssignmentContent"`
	IncludeRubric            bool                             `json:"includeRubric"`
	Status                   string                           `json:"status"`
	AutoGradeNew             *bool                            `json:"autoGradeNew"`
	ModelID                  *string                          `json:"modelId"`
	WorkflowGraph            *gradingagentsvc.WorkflowGraph `json:"workflowGraph"`
}

func (d Deps) handlePutGraderAgentConfig() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		courseCode, viewer, ok := d.requireGraderAgentAccess(w, r)
		if !ok {
			return
		}
		itemID, err := uuid.Parse(chi.URLParam(r, "item_id"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid assignment id.")
			return
		}
		cid, assignRow, ok := d.loadAssignmentForSubmissions(w, r, courseCode, itemID)
		if !ok || assignRow == nil || cid == nil {
			return
		}
		payload, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Could not read body.")
			return
		}
		var body putGraderAgentConfigBody
		if err := json.Unmarshal(payload, &body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		prompt := strings.TrimSpace(body.Prompt)
		includeContent := body.IncludeAssignmentContent
		includeRubric := body.IncludeRubric
		status := gradingagentrepo.StatusDraft
		switch strings.ToLower(strings.TrimSpace(body.Status)) {
		case "accepted":
			status = gradingagentrepo.StatusAccepted
		case "archived":
			status = gradingagentrepo.StatusArchived
		}
		var workflowGraphBytes []byte
		if body.WorkflowGraph != nil {
			if status == gradingagentrepo.StatusAccepted {
				if err := gradingagentsvc.ValidateWorkflowGraph(body.WorkflowGraph); err != nil {
					writeGraderAgentValidationError(w, err)
					return
				}
			} else if err := gradingagentsvc.ValidateWorkflowGraphForPersistence(body.WorkflowGraph); err != nil {
				writeGraderAgentValidationError(w, err)
				return
			}
			derivedPrompt, derivedContent, derivedRubric, derivedModel := gradingagentsvc.DeriveLegacyFields(body.WorkflowGraph)
			if derivedPrompt != "" {
				prompt = derivedPrompt
			}
			includeContent = derivedContent
			includeRubric = derivedRubric
			raw, marshalErr := gradingagentsvc.WorkflowGraphToJSON(body.WorkflowGraph)
			if marshalErr != nil {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid workflow graph.")
				return
			}
			workflowGraphBytes = raw
			if derivedModel != nil && (body.ModelID == nil || strings.TrimSpace(*body.ModelID) == "") {
				body.ModelID = derivedModel
			}
			if status == gradingagentrepo.StatusDraft {
				prompt = gradingagentsvc.PersistencePrompt(body.WorkflowGraph, prompt)
			}
		}
		if prompt == "" {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Prompt is required.")
			return
		}
		autoGrade := false
		if body.AutoGradeNew != nil {
			autoGrade = *body.AutoGradeNew
		}
		explicitModel := ""
		if body.ModelID != nil {
			explicitModel = strings.TrimSpace(*body.ModelID)
		}
		resolvedModel, modelErr := d.resolveGraderAgentModelID(r.Context(), viewer, explicitModel, nil)
		if modelErr != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, gradingagentsvc.UserFacingScoreError(modelErr))
			return
		}
		modelID := resolvedModel
		cfg, err := gradingagentrepo.UpsertConfig(r.Context(), d.Pool, gradingagentrepo.UpsertConfigInput{
			CourseID:                 *cid,
			ModuleItemID:             itemID,
			Status:                   status,
			Prompt:                   prompt,
			IncludeAssignmentContent: includeContent,
			IncludeRubric:            includeRubric,
			ModelID:                  &modelID,
			WorkflowGraph:            workflowGraphBytes,
			AutoGradeNew:             autoGrade,
			CreatedBy:                viewer,
		})
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to save grader agent.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{"config": graderAgentConfigToJSON(cfg)})
	}
}

type dryRunGraderAgentBody struct {
	Prompt                   string                           `json:"prompt"`
	IncludeAssignmentContent bool                             `json:"includeAssignmentContent"`
	IncludeRubric            bool                             `json:"includeRubric"`
	SubmissionID             string                           `json:"submissionId"`
	ModelID                  *string                          `json:"modelId"`
	WorkflowGraph            *gradingagentsvc.WorkflowGraph `json:"workflowGraph"`
}

func (d Deps) handlePostGraderAgentDryRun() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		courseCode, viewer, ok := d.requireGraderAgentAccess(w, r)
		if !ok {
			return
		}
		itemID, err := uuid.Parse(chi.URLParam(r, "item_id"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid assignment id.")
			return
		}
		cid, assignRow, ok := d.loadAssignmentForSubmissions(w, r, courseCode, itemID)
		if !ok || assignRow == nil || cid == nil {
			return
		}
		payload, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Could not read body.")
			return
		}
		var body dryRunGraderAgentBody
		if err := json.Unmarshal(payload, &body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		submissionID, err := uuid.Parse(strings.TrimSpace(body.SubmissionID))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid submission id.")
			return
		}
		subRow, err := moduleassignmentsubmissions.GetByIDForCourse(r.Context(), d.Pool, *cid, submissionID)
		if err != nil || subRow == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Submission not found.")
			return
		}
		if subRow.AttachmentFileID == nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "This submission has no readable text attachment. Upload a text file to grade with the agent.")
			return
		}
		svc := d.gradingAgentService()
		submissions, err := svc.LoadSubmissionMarkdownsForSubmission(r.Context(), courseCode, subRow)
		if err != nil {
			log.Printf("grading-agent dry-run: LoadSubmissionMarkdowns course=%s submission=%s file=%s err=%v",
				courseCode, submissionID, subRow.AttachmentFileID, err)
			msg := "Could not read submission content."
			switch {
			case strings.Contains(err.Error(), "empty submission text"):
				msg = "Submission file is empty. Use a text-based file the agent can read."
			case strings.Contains(err.Error(), "no submission text"):
				msg = "This submission has no readable text attachment."
			case strings.Contains(err.Error(), "read submission file"):
				msg = "Submission file could not be loaded. Re-upload the file or check storage configuration."
			case strings.Contains(err.Error(), "submission file not found"):
				msg = "Submission attachment record is missing from the course files table."
			}
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, msg)
			return
		}
		submissionText := gradingagentsvc.JoinSubmissions(submissions)
		explicitModel := ""
		if body.ModelID != nil {
			explicitModel = strings.TrimSpace(*body.ModelID)
		}
		var scoreReq gradingagentsvc.ScoreRequest
		var contentItemID string
		var rubricItemID string
		var workflowGraph *gradingagentsvc.WorkflowGraph
		var promptNodeID string
		governancePrompt := strings.TrimSpace(body.Prompt)
		if body.WorkflowGraph != nil {
			compiled, compileErr := gradingagentsvc.CompileWorkflowGraph(body.WorkflowGraph, submissionText)
			if compileErr != nil {
				writeGraderAgentValidationError(w, compileErr)
				return
			}
			scoreReq = compiled.ScoreRequest
			contentItemID = compiled.ContentItemID
			rubricItemID = compiled.RubricItemID
			workflowGraph = body.WorkflowGraph
			promptNodeID = compiled.GradeSource
			governancePrompt = scoreReq.InstructorPrompt
			if compiled.ScoreRequest.ModelID != "" {
				explicitModel = compiled.ScoreRequest.ModelID
			}
		} else {
			scoreReq = gradingagentsvc.ScoreRequest{
				InstructorPrompt:         strings.TrimSpace(body.Prompt),
				IncludeAssignmentContent: body.IncludeAssignmentContent,
				IncludeRubric:            body.IncludeRubric,
				SubmissionText:           submissionText,
			}
		}
		modelID, modelErr := d.resolveGraderAgentModelID(r.Context(), viewer, explicitModel, nil)
		if modelErr != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, gradingagentsvc.UserFacingScoreError(modelErr))
			return
		}
		scoreReq.ModelID = modelID
		if !d.enforceAIGateway(w, r, viewer, aigateway.FeatureGraderAgent, modelID, gradingagentsvc.ContentHashInput(governancePrompt, submissionText)) {
			return
		}
		if d.openRouterClient() == nil || strings.TrimSpace(d.effectiveConfig().OpenRouterAPIKey) == "" {
			apierr.WriteJSON(w, http.StatusServiceUnavailable, apierr.CodeInternal, "AI provider is not configured.")
			return
		}
		contentRow, contentErr := d.assignmentRowForActivitySource(r.Context(), *cid, itemID, assignRow, contentItemID)
		if contentErr != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, contentErr.Error())
			return
		}
		rubricRow, rubricErr := d.assignmentRowForActivitySource(r.Context(), *cid, itemID, assignRow, rubricItemID)
		if rubricErr != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, rubricErr.Error())
			return
		}
		rubric, _ := gradingagentsvc.ParseAssignmentRubric(rubricRow)
		scoreReq.AssignmentMarkdown = contentRow.Markdown
		scoreReq.Rubric = rubric
		scoreReq.MaxPoints = gradingagentsvc.MaxPointsFromAssignment(assignRow)
		if workflowGraph != nil && strings.TrimSpace(promptNodeID) != "" {
			scoreReq.InstructorPrompt = gradingagentsvc.SubstituteWorkflowPromptVariables(
				workflowGraph,
				promptNodeID,
				scoreReq.InstructorPrompt,
				gradingagentsvc.PromptVariableContext{
					Submissions:     submissions,
					ContentMarkdown: contentRow.Markdown,
					Rubric:          rubric,
				},
			)
		}
		result, err := svc.Score(r.Context(), scoreReq)
		if err != nil {
			log.Printf("grading-agent dry-run: Score course=%s submission=%s err=%v", courseCode, submissionID, err)
			apierr.WriteJSON(w, http.StatusBadGateway, apierr.CodeAiGenerationFailed, gradingagentsvc.UserFacingScoreError(err))
			return
		}
		d.recordAIUsage(r.Context(), AIUsageMeta{
			UserID: viewer, CourseCode: courseCode, Feature: aigateway.FeatureGraderAgent, Model: result.ModelID,
		}, openrouterUsageFromScore(result), true)
		d.logAIInferenceAllowed(r, viewer, aigateway.FeatureGraderAgent, result.ModelID, gradingagentsvc.ContentHashInput(governancePrompt, submissionText), aigateway.Decision{Allowed: true, OptInConfirmed: true})

		cfg, _ := gradingagentrepo.GetConfigByItem(r.Context(), d.Pool, itemID)
		var configID uuid.UUID
		savePrompt := governancePrompt
		saveIncludeContent := scoreReq.IncludeAssignmentContent
		saveIncludeRubric := scoreReq.IncludeRubric
		if cfg != nil {
			configID = cfg.ID
		} else {
			saved, saveErr := gradingagentrepo.UpsertConfig(r.Context(), d.Pool, gradingagentrepo.UpsertConfigInput{
				CourseID: *cid, ModuleItemID: itemID, Status: gradingagentrepo.StatusDraft,
				Prompt: savePrompt, IncludeAssignmentContent: saveIncludeContent,
				IncludeRubric: saveIncludeRubric, CreatedBy: viewer,
			})
			if saveErr == nil && saved != nil {
				configID = saved.ID
			}
		}
		if configID != uuid.Nil {
			comment := result.Output.Comment
			conf := result.Output.Confidence
			pt := result.PromptTokens
			ct := result.CompletionTokens
			cost := result.CostUSD
			model := result.ModelID
			pts := result.Output.TotalPoints
			_, _ = gradingagentrepo.InsertResult(r.Context(), d.Pool, gradingagentrepo.InsertResultInput{
				ConfigID: configID, SubmissionID: submissionID, IsDryRun: true,
				SuggestedPoints: &pts, Comment: &comment, Confidence: &conf,
				Status: gradingagentrepo.ItemSuggested, ModelID: &model,
				PromptTokens: &pt, CompletionTokens: &ct, CostUSD: &cost,
			})
		}

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"suggestedPoints":  result.Output.TotalPoints,
			"rubricScores":     result.Output.RubricScores,
			"comment":          result.Output.Comment,
			"confidence":       result.Output.Confidence,
			"promptTokens":     result.PromptTokens,
			"completionTokens": result.CompletionTokens,
		})
	}
}

type postGraderAgentRunBody struct {
	Scope        string  `json:"scope"`
	SubmissionID string  `json:"submissionId"`
	Overwrite    bool    `json:"overwrite"`
	AuthoredVia  *string `json:"authoredVia"`
}

func (d Deps) handlePostGraderAgentRun() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		courseCode, viewer, ok := d.requireGraderAgentAccess(w, r)
		if !ok {
			return
		}
		itemID, err := uuid.Parse(chi.URLParam(r, "item_id"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid assignment id.")
			return
		}
		cid, _, ok := d.loadAssignmentForSubmissions(w, r, courseCode, itemID)
		if !ok || cid == nil {
			return
		}
		cfg, err := gradingagentrepo.GetConfigByItem(r.Context(), d.Pool, itemID)
		if err != nil || cfg == nil || cfg.Status != gradingagentrepo.StatusAccepted {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Accept the grading agent before running.")
			return
		}
		payload, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Could not read body.")
			return
		}
		var body postGraderAgentRunBody
		if err := json.Unmarshal(payload, &body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		scope := gradingagentrepo.RunScope(strings.ToLower(strings.TrimSpace(body.Scope)))
		submissions, runScope, err := d.resolveGraderAgentSubmissions(r.Context(), *cid, itemID, scope, body.SubmissionID, body.Overwrite)
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, err.Error())
			return
		}
		if len(submissions) == 0 {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "No submissions with readable file attachments matched this scope.")
			return
		}
		initiatedBy := viewer
		var authoredVia *string
		if body.AuthoredVia != nil {
			v := strings.TrimSpace(*body.AuthoredVia)
			if v == "canvas" || v == "form" {
				authoredVia = &v
			}
		}
		run, err := gradingagentrepo.CreateRun(r.Context(), d.Pool, cfg.ID, runScope, &initiatedBy, authoredVia, len(submissions))
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to start run.")
			return
		}
		if d.GradingAgentQueue == nil {
			apierr.WriteJSON(w, http.StatusServiceUnavailable, apierr.CodeInternal, "Grading agent queue is not configured.")
			return
		}
		_ = gradingagentrepo.MarkRunRunning(r.Context(), d.Pool, run.ID)
		for _, sub := range submissions {
			msg := gradingagentqueue.QueueMessage{
				RunID: run.ID, ConfigID: cfg.ID, SubmissionID: sub.ID,
				CourseID: *cid, ItemID: itemID, CourseCode: courseCode,
			}
			if pubErr := d.GradingAgentQueue.Publish(r.Context(), msg); pubErr != nil {
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to enqueue run.")
				return
			}
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusAccepted)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"runId":      run.ID.String(),
			"totalCount": run.TotalCount,
		})
	}
}

func (d Deps) resolveGraderAgentSubmissions(
	ctx context.Context,
	courseID, itemID uuid.UUID,
	scope gradingagentrepo.RunScope,
	submissionID string,
	overwrite bool,
) ([]moduleassignmentsubmissions.SubmissionRow, gradingagentrepo.RunScope, error) {
	switch scope {
	case gradingagentrepo.RunScopeCurrent:
		sid, err := uuid.Parse(strings.TrimSpace(submissionID))
		if err != nil {
			return nil, scope, errInvalidScope("submissionId is required for current scope")
		}
		sub, err := moduleassignmentsubmissions.GetByIDForCourse(ctx, d.Pool, courseID, sid)
		if err != nil || sub == nil {
			return nil, scope, errInvalidScope("submission not found")
		}
		if sub.AttachmentFileID == nil {
			return nil, scope, errInvalidScope("submission has no file attachment")
		}
		return []moduleassignmentsubmissions.SubmissionRow{*sub}, scope, nil
	case gradingagentrepo.RunScopeUngraded:
		rows, err := moduleassignmentsubmissions.ListForAssignment(ctx, d.Pool, courseID, itemID, moduleassignmentsubmissions.GradedFilterUngraded)
		return gradableSubmissionsForAgent(rows), scope, err
	case gradingagentrepo.RunScopeAll:
		if !overwrite {
			return nil, scope, errInvalidScope("overwrite confirmation required for all scope")
		}
		rows, err := moduleassignmentsubmissions.ListForAssignment(ctx, d.Pool, courseID, itemID, moduleassignmentsubmissions.GradedFilterAll)
		return gradableSubmissionsForAgent(rows), scope, err
	default:
		return nil, scope, errInvalidScope("invalid scope")
	}
}

// gradableSubmissionsForAgent keeps only assignment submissions that have a stored file to read.
func gradableSubmissionsForAgent(rows []moduleassignmentsubmissions.SubmissionRow) []moduleassignmentsubmissions.SubmissionRow {
	if len(rows) == 0 {
		return rows
	}
	out := make([]moduleassignmentsubmissions.SubmissionRow, 0, len(rows))
	for _, row := range rows {
		if row.AttachmentFileID != nil {
			out = append(out, row)
		}
	}
	return out
}

type invalidScopeError string

func errInvalidScope(msg string) error { return invalidScopeError(msg) }
func (e invalidScopeError) Error() string { return string(e) }

type patchGraderAgentResultBody struct {
	Status string  `json:"status"`
	Reason *string `json:"reason"`
}

func (d Deps) handlePatchGraderAgentResult() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, _, ok := d.requireGraderAgentAccess(w, r)
		if !ok {
			return
		}
		itemID, err := uuid.Parse(chi.URLParam(r, "item_id"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid assignment id.")
			return
		}
		resultID, err := uuid.Parse(chi.URLParam(r, "result_id"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid result id.")
			return
		}
		cfg, err := gradingagentrepo.GetConfigByItem(r.Context(), d.Pool, itemID)
		if err != nil || cfg == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Grader agent not found.")
			return
		}
		var body patchGraderAgentResultBody
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		status := strings.TrimSpace(body.Status)
		switch status {
		case string(gradingagentrepo.ItemApplied), string(gradingagentrepo.ItemOverridden), string(gradingagentrepo.ItemSkipped):
		default:
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Status must be applied, overridden, or skipped.")
			return
		}
		updated, err := gradingagentrepo.UpdateResultStatus(r.Context(), d.Pool, resultID, gradingagentrepo.ItemStatus(status), body.Reason)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to update result.")
			return
		}
		if updated == nil || updated.ConfigID != cfg.ID {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Result not found.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":           updated.ID.String(),
			"submissionId": updated.SubmissionID.String(),
			"status":       string(updated.Status),
		})
	}
}

func (d Deps) handleGetGraderAgentRun() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		courseCode, _, ok := d.requireGraderAgentAccess(w, r)
		if !ok {
			return
		}
		itemID, err := uuid.Parse(chi.URLParam(r, "item_id"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid assignment id.")
			return
		}
		runID, err := uuid.Parse(chi.URLParam(r, "run_id"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid run id.")
			return
		}
		cfg, err := gradingagentrepo.GetConfigByItem(r.Context(), d.Pool, itemID)
		if err != nil || cfg == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Grader agent not found.")
			return
		}
		run, err := gradingagentrepo.GetRun(r.Context(), d.Pool, runID)
		if err != nil || run == nil || run.ConfigID != cfg.ID {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Run not found.")
			return
		}
		results, err := gradingagentrepo.ListResultsForRun(r.Context(), d.Pool, runID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load run.")
			return
		}
		resultJSON := make([]map[string]any, 0, len(results))
		for _, res := range results {
			entry := map[string]any{
				"id":           res.ID.String(),
				"submissionId": res.SubmissionID.String(),
				"status":       string(res.Status),
			}
			if res.SuggestedPoints != nil {
				entry["suggestedPoints"] = *res.SuggestedPoints
			}
			if res.Comment != nil {
				entry["comment"] = *res.Comment
			}
			if res.Confidence != nil {
				entry["confidence"] = *res.Confidence
			}
			if res.Error != nil {
				entry["error"] = *res.Error
			}
			if res.FlagReason != nil {
				entry["flagReason"] = *res.FlagReason
			}
			if res.FlagPriority != nil {
				entry["flagPriority"] = *res.FlagPriority
			}
			if res.HeldReason != nil {
				entry["heldReason"] = *res.HeldReason
			}
			if res.HeldAt != nil {
				entry["heldAt"] = res.HeldAt.UTC().Format("2006-01-02T15:04:05.000000Z")
			}
			if res.HeldQueue != nil {
				entry["heldQueue"] = *res.HeldQueue
			}
			resultJSON = append(resultJSON, entry)
		}
		_ = courseCode
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status":         run.Status,
			"totalCount":     run.TotalCount,
			"completedCount": run.CompletedCount,
			"failedCount":    run.FailedCount,
			"results":        resultJSON,
		})
	}
}

type postGraderAgentTemplateBody struct {
	Name                     string                           `json:"name"`
	Prompt                   string                           `json:"prompt"`
	IncludeAssignmentContent bool                             `json:"includeAssignmentContent"`
	IncludeRubric            bool                             `json:"includeRubric"`
	WorkflowGraph            *gradingagentsvc.WorkflowGraph `json:"workflowGraph"`
}

func graderAgentTemplateToJSON(row *gradingagentrepo.TemplateRow) map[string]any {
	if row == nil {
		return nil
	}
	out := map[string]any{
		"id":                       row.ID.String(),
		"name":                     row.Name,
		"prompt":                   row.Prompt,
		"includeAssignmentContent": row.IncludeAssignmentContent,
		"includeRubric":            row.IncludeRubric,
		"createdAt":                row.CreatedAt.UTC().Format("2006-01-02T15:04:05.000000Z"),
		"updatedAt":                row.UpdatedAt.UTC().Format("2006-01-02T15:04:05.000000Z"),
	}
	if g, err := gradingagentsvc.EffectiveWorkflowGraph(row.WorkflowGraph, row.Prompt, row.IncludeAssignmentContent, row.IncludeRubric); err == nil && g != nil {
		out["workflowGraph"] = g
	}
	return out
}

func (d Deps) handleGetGraderAgentTemplate() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		courseCode, _, ok := d.requireGraderAgentAccess(w, r)
		if !ok {
			return
		}
		templateID, err := uuid.Parse(chi.URLParam(r, "template_id"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid template id.")
			return
		}
		cid, found, err := d.courseIDFromCode(r.Context(), courseCode)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load course.")
			return
		}
		if !found {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Course not found.")
			return
		}
		tmpl, err := gradingagentrepo.GetTemplateByCourseAndID(r.Context(), d.Pool, cid, templateID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Template not found.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{"template": graderAgentTemplateToJSON(tmpl)})
	}
}

func (d Deps) handlePutGraderAgentTemplate() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		courseCode, _, ok := d.requireGraderAgentAccess(w, r)
		if !ok {
			return
		}
		templateID, err := uuid.Parse(chi.URLParam(r, "template_id"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid template id.")
			return
		}
		cid, found, err := d.courseIDFromCode(r.Context(), courseCode)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load course.")
			return
		}
		if !found {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Course not found.")
			return
		}
		payload, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Could not read body.")
			return
		}
		var body postGraderAgentTemplateBody
		if err := json.Unmarshal(payload, &body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		name := strings.TrimSpace(body.Name)
		if name == "" {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Template name is required.")
			return
		}
		if body.WorkflowGraph == nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Workflow graph is required.")
			return
		}
		if err := gradingagentsvc.ValidateWorkflowGraphForPersistence(body.WorkflowGraph); err != nil {
			writeGraderAgentValidationError(w, err)
			return
		}
		prompt := strings.TrimSpace(body.Prompt)
		derivedPrompt, includeContent, includeRubric, _ := gradingagentsvc.DeriveLegacyFields(body.WorkflowGraph)
		if derivedPrompt != "" {
			prompt = derivedPrompt
		}
		prompt = gradingagentsvc.PersistencePrompt(body.WorkflowGraph, prompt)
		if prompt == "" {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Prompt is required.")
			return
		}
		raw, marshalErr := gradingagentsvc.WorkflowGraphToJSON(body.WorkflowGraph)
		if marshalErr != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid workflow graph.")
			return
		}
		tmpl, err := gradingagentrepo.UpdateTemplate(r.Context(), d.Pool, cid, templateID, gradingagentrepo.UpdateTemplateInput{
			Name:                     name,
			Prompt:                   prompt,
			IncludeAssignmentContent: includeContent,
			IncludeRubric:            includeRubric,
			WorkflowGraph:            raw,
		})
		if err != nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Template not found.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{"template": graderAgentTemplateToJSON(tmpl)})
	}
}

func (d Deps) handleListGraderAgentTemplates() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		courseCode, _, ok := d.requireGraderAgentAccess(w, r)
		if !ok {
			return
		}
		cid, found, err := d.courseIDFromCode(r.Context(), courseCode)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load course.")
			return
		}
		if !found {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Course not found.")
			return
		}
		rows, err := gradingagentrepo.ListTemplatesByCourse(r.Context(), d.Pool, cid)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load templates.")
			return
		}
		templates := make([]map[string]any, 0, len(rows))
		for _, row := range rows {
			templates = append(templates, map[string]any{
				"id":        row.ID.String(),
				"name":      row.Name,
				"isBuiltin": gradingagentsvc.IsDefaultTemplateName(row.Name),
				"updatedAt": row.UpdatedAt.UTC().Format("2006-01-02T15:04:05.000000Z"),
			})
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{"templates": templates})
	}
}

func (d Deps) handleDeleteGraderAgentTemplate() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		courseCode, _, ok := d.requireGraderAgentAccess(w, r)
		if !ok {
			return
		}
		templateID, err := uuid.Parse(chi.URLParam(r, "template_id"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid template id.")
			return
		}
		cid, found, err := d.courseIDFromCode(r.Context(), courseCode)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load course.")
			return
		}
		if !found {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Course not found.")
			return
		}
		tmpl, err := gradingagentrepo.GetTemplateByCourseAndID(r.Context(), d.Pool, cid, templateID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Template not found.")
			return
		}
		if gradingagentsvc.IsDefaultTemplateName(tmpl.Name) {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "Built-in templates cannot be deleted.")
			return
		}
		deleted, err := gradingagentrepo.DeleteTemplate(r.Context(), d.Pool, cid, templateID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to delete template.")
			return
		}
		if !deleted {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Template not found.")
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func (d Deps) handlePostGraderAgentTemplate() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		courseCode, viewer, ok := d.requireGraderAgentAccess(w, r)
		if !ok {
			return
		}
		cid, found, err := d.courseIDFromCode(r.Context(), courseCode)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load course.")
			return
		}
		if !found {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Course not found.")
			return
		}
		payload, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Could not read body.")
			return
		}
		var body postGraderAgentTemplateBody
		if err := json.Unmarshal(payload, &body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		name := strings.TrimSpace(body.Name)
		if name == "" {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Template name is required.")
			return
		}
		if body.WorkflowGraph == nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Workflow graph is required.")
			return
		}
		if err := gradingagentsvc.ValidateWorkflowGraphForPersistence(body.WorkflowGraph); err != nil {
			writeGraderAgentValidationError(w, err)
			return
		}
		prompt := strings.TrimSpace(body.Prompt)
		derivedPrompt, includeContent, includeRubric, _ := gradingagentsvc.DeriveLegacyFields(body.WorkflowGraph)
		if derivedPrompt != "" {
			prompt = derivedPrompt
		}
		prompt = gradingagentsvc.PersistencePrompt(body.WorkflowGraph, prompt)
		if prompt == "" {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Prompt is required.")
			return
		}
		raw, marshalErr := gradingagentsvc.WorkflowGraphToJSON(body.WorkflowGraph)
		if marshalErr != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid workflow graph.")
			return
		}
		tmpl, err := gradingagentrepo.CreateTemplate(r.Context(), d.Pool, gradingagentrepo.CreateTemplateInput{
			CourseID:                 cid,
			Name:                     name,
			Prompt:                   prompt,
			IncludeAssignmentContent: includeContent,
			IncludeRubric:            includeRubric,
			WorkflowGraph:            raw,
			CreatedBy:                viewer,
		})
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to save template.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{"template": graderAgentTemplateToJSON(tmpl)})
	}
}

func (d Deps) handlePostGraderAgentRegradeRequest() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.graderAgentEnabled() {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Grader agent is not enabled.")
			return
		}
		courseCode, viewer, ok := d.requireCourseAccess(w, r)
		if !ok {
			return
		}
		itemID, err := uuid.Parse(chi.URLParam(r, "item_id"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid assignment id.")
			return
		}
		cid, err := course.GetIDByCourseCode(r.Context(), d.Pool, courseCode)
		if err != nil || cid == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Course not found.")
			return
		}
		cell, err := coursegrades.GetCell(r.Context(), d.Pool, *cid, viewer, itemID)
		if err != nil || cell == nil || !cell.GradedByAI || cell.PostedAt == nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "No posted AI-graded feedback to appeal.")
			return
		}
		actionURL := "/courses/" + courseCode + "/assignments/" + itemID.String()
		rows, qerr := d.Pool.Query(r.Context(), `
SELECT ce.user_id
FROM course.course_enrollments ce
INNER JOIN course.courses c ON c.id = ce.course_id
INNER JOIN course.enrollment_roles er ON er.role_key = ce.role AND er.is_staff = true
WHERE c.course_code = $1 AND ce.status = 'active'
`, courseCode)
		if qerr != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to notify instructors.")
			return
		}
		defer rows.Close()
		for rows.Next() {
			var staffID uuid.UUID
			if scanErr := rows.Scan(&staffID); scanErr != nil {
				continue
			}
			_, _ = notificationsinbox.Insert(r.Context(), d.Pool, staffID, "grader_agent_regrade_request",
				"Human re-grade requested",
				"A student requested human review of AI-drafted feedback.",
				actionURL,
			)
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
	}
}

func openrouterUsageFromScore(result gradingagentsvc.ScoreResult) openrouter.UsageInfo {
	return openrouter.UsageInfo{
		PromptTokens:     result.PromptTokens,
		CompletionTokens: result.CompletionTokens,
		TotalTokens:      result.PromptTokens + result.CompletionTokens,
		CostUSD:          result.CostUSD,
	}
}