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
	"github.com/lextures/lextures/server/internal/gradingredaction"
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
)

func (d Deps) graderAgentEnabled() bool {
	return d.effectiveConfig().GraderAgentEnabled
}

func (d Deps) graderAgentReviewInboxEnabled() bool {
	return d.effectiveConfig().GraderAgentReviewInboxEnabled
}

func (d Deps) graderAgentTextEntryGradingEnabled() bool {
	return d.effectiveConfig().GraderAgentTextEntryGradingEnabled
}

func (d Deps) graderAgentVisionGradingEnabled() bool {
	return d.effectiveConfig().GraderAgentVisionGradingEnabled
}

func (d Deps) graderAgentRunFiltersEnabled() bool {
	return d.effectiveConfig().GraderAgentRunFiltersEnabled
}

func (d Deps) graderAgentCostEstimateEnabled() bool {
	return d.effectiveConfig().GraderAgentCostEstimateEnabled
}

func (d Deps) graderAgentCancelRunEnabled() bool {
	return d.effectiveConfig().GraderAgentCancelRunEnabled
}

func (d Deps) requireGraderAgentReviewInboxAccess(w http.ResponseWriter, r *http.Request) (courseCode string, viewer uuid.UUID, ok bool) {
	if !d.graderAgentReviewInboxEnabled() {
		apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Grader agent review inbox is not enabled.")
		return "", uuid.Nil, false
	}
	return d.requireGraderAgentAccess(w, r)
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
	postPolicy := row.PostPolicy
	if postPolicy == "" || postPolicy == "unposted" {
		postPolicy = "draft"
	}
	out := map[string]any{
		"id":                       row.ID.String(),
		"prompt":                   row.Prompt,
		"includeAssignmentContent": row.IncludeAssignmentContent,
		"includeRubric":            row.IncludeRubric,
		"status":                   string(row.Status),
		"autoGradeNew":             row.AutoGradeNew,
		"postPolicy":               postPolicy,
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
		reviewCounts := map[uuid.UUID]int{}
		if d.graderAgentReviewInboxEnabled() {
			configIDs := make([]uuid.UUID, 0, len(rows))
			for _, row := range rows {
				configIDs = append(configIDs, row.ID)
			}
			if counts, countErr := gradingagentrepo.CountReviewQueueByConfigs(r.Context(), d.Pool, configIDs); countErr == nil {
				reviewCounts = counts
			}
		}
		agents := make([]map[string]any, 0, len(rows))
		for _, row := range rows {
			entry := map[string]any{
				"id":                 row.ID.String(),
				"itemId":             row.ModuleItemID.String(),
				"itemKind":           row.ItemKind,
				"assignmentTitle":    row.AssignmentTitle,
				"assignmentArchived": row.AssignmentArchived,
				"status":             string(row.Status),
				"autoGradeNew":       row.AutoGradeNew,
				"hasWorkflowGraph":   row.HasWorkflowGraph,
				"updatedAt":          row.UpdatedAt.UTC().Format("2006-01-02T15:04:05.000000Z"),
			}
			if d.graderAgentReviewInboxEnabled() {
				entry["reviewCount"] = reviewCounts[row.ID]
			}
			agents = append(agents, entry)
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
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid item id.")
			return
		}
		if _, ok := d.loadGradingAgentModuleItem(w, r, courseCode, itemID); !ok {
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
	PostPolicy               string                           `json:"postPolicy"`
	ConfidenceFloor          *float64                         `json:"confidenceFloor"`
	ModelID                  *string                          `json:"modelId"`
	WorkflowGraph            *gradingagentsvc.WorkflowGraph `json:"workflowGraph"`
}

func (d Deps) handleDeleteGraderAgentConfig() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		courseCode, _, ok := d.requireGraderAgentAccess(w, r)
		if !ok {
			return
		}
		itemID, err := uuid.Parse(chi.URLParam(r, "item_id"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid item id.")
			return
		}
		item, ok := d.loadGradingAgentModuleItem(w, r, courseCode, itemID)
		if !ok || item == nil {
			return
		}
		deleted, err := gradingagentrepo.DeleteConfig(r.Context(), d.Pool, item.CourseID, itemID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to delete grading agent.")
			return
		}
		if !deleted {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Grading agent not found.")
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func (d Deps) handlePutGraderAgentConfig() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		courseCode, viewer, ok := d.requireGraderAgentAccess(w, r)
		if !ok {
			return
		}
		itemID, err := uuid.Parse(chi.URLParam(r, "item_id"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid item id.")
			return
		}
		item, ok := d.loadGradingAgentModuleItem(w, r, courseCode, itemID)
		if !ok || item == nil {
			return
		}
		cid := &item.CourseID
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
		var payloadKeys map[string]json.RawMessage
		_ = json.Unmarshal(payload, &payloadKeys)
		_, confidenceFloorInBody := payloadKeys["confidenceFloor"]
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
		postPolicy := "draft"
		if strings.TrimSpace(body.PostPolicy) == "auto_post" {
			postPolicy = "auto_post"
		}
		var confidenceFloor *float64
		if confidenceFloorInBody {
			if body.ConfidenceFloor != nil {
				floor := *body.ConfidenceFloor
				if floor < 0 || floor > 1 {
					apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "confidenceFloor must be between 0 and 1.")
					return
				}
				if floor > 0 {
					confidenceFloor = &floor
				}
			}
		} else {
			existing, loadErr := gradingagentrepo.GetConfigByItem(r.Context(), d.Pool, itemID)
			if loadErr != nil {
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load grader agent.")
				return
			}
			if existing != nil {
				confidenceFloor = existing.ConfidenceFloor
			}
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
			PostPolicy:               postPolicy,
			ConfidenceFloor:          confidenceFloor,
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

type postGraderAgentRunBody struct {
	Scope        string                     `json:"scope"`
	Mode         string                     `json:"mode"`
	SubmissionID string                     `json:"submissionId"`
	Overwrite    bool                       `json:"overwrite"`
	AuthoredVia  *string                    `json:"authoredVia"`
	Filter       *graderAgentRunFilterBody  `json:"filter"`
	BudgetUSD    *float64                   `json:"budgetUsd"`
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
		var runFilter *gradingagentrepo.RunFilter
		var filterMeta *graderAgentRunFilterContext
		if d.graderAgentRunFiltersEnabled() && body.Filter != nil {
			parsed, parseErr := parseGraderAgentRunFilterBody(body.Filter)
			if parseErr != nil {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, parseErr.Error())
				return
			}
			if parsed != nil && !parsed.IsEmpty() {
				meta, valErr := d.validateGraderAgentRunFilter(r.Context(), *cid, itemID, courseCode, viewer, parsed)
				if valErr != nil {
					apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, valErr.Error())
					return
				}
				runFilter = parsed
				filterMeta = meta
			}
		}
		submissions, runScope, err := d.resolveGraderAgentSubmissions(
			r.Context(), courseCode, *cid, itemID, viewer, scope, body.SubmissionID, body.Overwrite, runFilter, d.graderAgentTextEntryGradingEnabled(),
		)
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, err.Error())
			return
		}
		if len(submissions) == 0 {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "No gradable submissions matched this scope.")
			return
		}
		targetSummary := formatGraderAgentRunTargetSummary(runScope, filterMeta, len(submissions))
		filterJSON, _ := runFilter.ToJSON()
		initiatedBy := viewer
		var authoredVia *string
		if body.AuthoredVia != nil {
			v := strings.TrimSpace(*body.AuthoredVia)
			if v == "canvas" || v == "form" {
				authoredVia = &v
			}
		}
		runMode := gradingagentrepo.RunModeApply
		if d.graderAgentSuggestModeEnabled() {
			switch strings.ToLower(strings.TrimSpace(body.Mode)) {
			case "apply":
				runMode = gradingagentrepo.RunModeApply
			default:
				runMode = gradingagentrepo.RunModeSuggest
			}
		}
		var budgetUSD *float64
		if d.graderAgentCostEstimateEnabled() && body.BudgetUSD != nil && *body.BudgetUSD > 0 {
			v := *body.BudgetUSD
			budgetUSD = &v
		}
		run, err := gradingagentrepo.CreateRun(r.Context(), d.Pool, cfg.ID, runScope, runMode, &initiatedBy, authoredVia, len(submissions), filterJSON, budgetUSD)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to start run.")
			return
		}
		if d.GradingAgentQueue == nil {
			apierr.WriteJSON(w, http.StatusServiceUnavailable, apierr.CodeInternal, "Grading agent queue is not configured.")
			return
		}
		_ = gradingagentrepo.MarkRunRunning(r.Context(), d.Pool, run.ID)
		queued := 0
		for _, sub := range submissions {
			msg := gradingagentqueue.QueueMessage{
				RunID: run.ID, ConfigID: cfg.ID, SubmissionID: sub.ID,
				CourseID: *cid, ItemID: itemID, CourseCode: courseCode,
			}
			if pubErr := d.GradingAgentQueue.Publish(r.Context(), msg); pubErr != nil {
				log.Printf("grading_agent: enqueue failed after %d/%d items for run %s: %v", queued, len(submissions), run.ID, pubErr)
				_ = gradingagentrepo.MarkRunFailed(context.Background(), d.Pool, run.ID)
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal,
					fmt.Sprintf("Enqueue failed after %d of %d items.", queued, len(submissions)))
				return
			}
			queued++
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusAccepted)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"runId":         run.ID.String(),
			"totalCount":    run.TotalCount,
			"queuedCount":   queued,
			"mode":          string(run.Mode),
			"targetSummary": targetSummary,
		})
	}
}

func (d Deps) resolveGraderAgentSubmissions(
	ctx context.Context,
	courseCode string,
	courseID, itemID uuid.UUID,
	viewer uuid.UUID,
	scope gradingagentrepo.RunScope,
	submissionID string,
	overwrite bool,
	runFilter *gradingagentrepo.RunFilter,
	textEntryEnabled bool,
) ([]moduleassignmentsubmissions.SubmissionRow, gradingagentrepo.RunScope, error) {
	listFilter := graderAgentListFilterFromRunFilter(runFilter)
	visibleSections, err := d.graderAgentVisibleSectionIDs(ctx, courseID, courseCode, viewer)
	if err != nil {
		return nil, scope, err
	}
	if len(visibleSections) > 0 {
		listFilter.VisibleSectionIDs = visibleSections
	}
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
		if !gradingagentsvc.SubmissionAttemptableForAgent(*sub, textEntryEnabled) {
			return nil, scope, errInvalidScope("submission has no gradable content")
		}
		if len(visibleSections) > 0 {
			rows, listErr := moduleassignmentsubmissions.ListForAssignmentFiltered(
				ctx, d.Pool, courseID, itemID, moduleassignmentsubmissions.GradedFilterAll,
				moduleassignmentsubmissions.ListFilter{
					SubmissionIDs:     []uuid.UUID{sid},
					VisibleSectionIDs: visibleSections,
				},
			)
			if listErr != nil || len(rows) == 0 {
				return nil, scope, errInvalidScope("submission not found")
			}
		}
		return []moduleassignmentsubmissions.SubmissionRow{*sub}, scope, nil
	case gradingagentrepo.RunScopeUngraded:
		rows, err := moduleassignmentsubmissions.ListForAssignmentFiltered(
			ctx, d.Pool, courseID, itemID, moduleassignmentsubmissions.GradedFilterUngraded, listFilter,
		)
		return gradableSubmissionsForAgent(rows, textEntryEnabled), scope, err
	case gradingagentrepo.RunScopeAll:
		if !overwrite {
			return nil, scope, errInvalidScope("overwrite confirmation required for all scope")
		}
		rows, err := moduleassignmentsubmissions.ListForAssignmentFiltered(
			ctx, d.Pool, courseID, itemID, moduleassignmentsubmissions.GradedFilterAll, listFilter,
		)
		return gradableSubmissionsForAgent(rows, textEntryEnabled), scope, err
	default:
		return nil, scope, errInvalidScope("invalid scope")
	}
}

// gradableSubmissionsForAgent keeps submissions with file attachments and/or typed text-entry body.
func gradableSubmissionsForAgent(rows []moduleassignmentsubmissions.SubmissionRow, textEntryEnabled bool) []moduleassignmentsubmissions.SubmissionRow {
	if len(rows) == 0 {
		return rows
	}
	out := make([]moduleassignmentsubmissions.SubmissionRow, 0, len(rows))
	for _, row := range rows {
		if gradingagentsvc.SubmissionAttemptableForAgent(row, textEntryEnabled) {
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
		_, viewer, ok := d.requireGraderAgentAccess(w, r)
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
		updated, err := gradingagentrepo.UpdateResultStatus(r.Context(), d.Pool, resultID, gradingagentrepo.ItemStatus(status), body.Reason, &viewer)
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
		usage, usageErr := gradingagentrepo.SumRunUsage(r.Context(), d.Pool, runID)
		if usageErr != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load run usage.")
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
			if res.InputModality != nil {
				entry["inputModality"] = *res.InputModality
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
		resp := map[string]any{
			"status":         run.Status,
			"totalCount":     run.TotalCount,
			"completedCount": run.CompletedCount,
			"failedCount":    run.FailedCount,
			"results":        resultJSON,
		}
		if run.BudgetUSD != nil {
			resp["budgetUsd"] = *run.BudgetUSD
		}
		for k, v := range runUsageToJSON(usage) {
			resp[k] = v
		}
		if run.CancelledAt != nil {
			resp["cancelledAt"] = run.CancelledAt.UTC().Format("2006-01-02T15:04:05.000000Z")
		}
		if run.CancelledBy != nil {
			resp["cancelledBy"] = run.CancelledBy.String()
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(resp)
	}
}

func (d Deps) handlePostGraderAgentCancelRun() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.graderAgentCancelRunEnabled() {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Grader agent run cancel is not enabled.")
			return
		}
		_, viewer, ok := d.requireGraderAgentAccess(w, r)
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
		switch run.Status {
		case gradingagentrepo.RunStatusQueued, gradingagentrepo.RunStatusRunning:
		default:
			apierr.WriteJSON(w, http.StatusConflict, apierr.CodeConflict, "Run cannot be cancelled.")
			return
		}
		cancelled, err := gradingagentrepo.CancelRun(r.Context(), d.Pool, runID, viewer)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to cancel run.")
			return
		}
		if !cancelled {
			apierr.WriteJSON(w, http.StatusConflict, apierr.CodeConflict, "Run cannot be cancelled.")
			return
		}
		gradingAgentRunStatusCacheInvalidate(runID)
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{"status": gradingagentrepo.RunStatusCancelled})
	}
}

func graderAgentReviewQueueItemToJSON(item gradingagentrepo.ReviewQueueItem, label string) map[string]any {
	entry := map[string]any{
		"id":             item.ID.String(),
		"submissionId":   item.SubmissionID.String(),
		"submissionLabel": label,
		"status":         string(item.Status),
		"createdAt":      item.CreatedAt.UTC().Format("2006-01-02T15:04:05.000000Z"),
	}
	if item.RunID != nil {
		entry["runId"] = item.RunID.String()
	}
	if item.RunCreatedAt != nil {
		entry["runCreatedAt"] = item.RunCreatedAt.UTC().Format("2006-01-02T15:04:05.000000Z")
	}
	if item.SuggestedPoints != nil {
		entry["suggestedPoints"] = *item.SuggestedPoints
	}
	if item.Comment != nil {
		entry["comment"] = *item.Comment
	}
	if item.Confidence != nil {
		entry["confidence"] = *item.Confidence
	}
	if item.FlagReason != nil {
		entry["flagReason"] = *item.FlagReason
	}
	if item.FlagPriority != nil {
		entry["flagPriority"] = *item.FlagPriority
	}
	if item.HeldReason != nil {
		entry["heldReason"] = *item.HeldReason
	}
	if item.HeldAt != nil {
		entry["heldAt"] = item.HeldAt.UTC().Format("2006-01-02T15:04:05.000000Z")
	}
	if item.HeldQueue != nil {
		entry["heldQueue"] = *item.HeldQueue
	}
	return entry
}

func (d Deps) submissionLabelsForGraderAgentReview(
	ctx context.Context,
	courseID, itemID uuid.UUID,
	assignRow *coursemoduleassignments.CourseItemAssignmentRow,
	submissionIDs []uuid.UUID,
) (map[uuid.UUID]string, error) {
	labels := make(map[uuid.UUID]string, len(submissionIDs))
	if len(submissionIDs) == 0 || d.Pool == nil || assignRow == nil {
		return labels, nil
	}
	rows, err := moduleassignmentsubmissions.ListForAssignment(ctx, d.Pool, courseID, itemID, moduleassignmentsubmissions.GradedFilterAll)
	if err != nil {
		return nil, err
	}
	subByID := make(map[uuid.UUID]moduleassignmentsubmissions.SubmissionRow, len(rows))
	for _, row := range rows {
		subByID[row.ID] = row
	}
	cfg := d.effectiveConfig()
	redact := gradingredaction.ShouldRedactSubmissionPiiForStaff(
		cfg.BlindGradingEnabled,
		assignRow.BlindGrading,
		assignRow.IdentitiesRevealedAt != nil,
	)
	userIDs := make([]uuid.UUID, 0, len(submissionIDs))
	for _, sid := range submissionIDs {
		if sub, ok := subByID[sid]; ok {
			userIDs = append(userIDs, sub.SubmittedBy)
		}
	}
	displayNames := map[uuid.UUID]string{}
	if !redact && len(userIDs) > 0 {
		displayNames, err = user.DisplayLabelsByIDs(ctx, d.Pool, userIDs)
		if err != nil {
			return nil, err
		}
	}
	blindRanks := map[uuid.UUID]int{}
	if redact {
		rosterIDs := make([]uuid.UUID, 0, len(rows))
		seen := map[uuid.UUID]struct{}{}
		for _, row := range rows {
			if _, ok := seen[row.SubmittedBy]; ok {
				continue
			}
			seen[row.SubmittedBy] = struct{}{}
			rosterIDs = append(rosterIDs, row.SubmittedBy)
		}
		blindRanks = gradingredaction.SubmissionRankByID(rosterIDs)
	}
	for _, sid := range submissionIDs {
		sub, ok := subByID[sid]
		if !ok {
			labels[sid] = sid.String()[:8]
			continue
		}
		if redact {
			labels[sid] = gradingredaction.BlindStudentLabel(blindRanks[sub.SubmittedBy])
			continue
		}
		label := strings.TrimSpace(displayNames[sub.SubmittedBy])
		if label == "" {
			label = sub.SubmittedBy.String()[:8]
		}
		labels[sid] = label
	}
	return labels, nil
}

func (d Deps) handleListGraderAgentRuns() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, _, ok := d.requireGraderAgentReviewInboxAccess(w, r)
		if !ok {
			return
		}
		itemID, err := uuid.Parse(chi.URLParam(r, "item_id"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid assignment id.")
			return
		}
		cfg, err := gradingagentrepo.GetConfigByItem(r.Context(), d.Pool, itemID)
		if err != nil || cfg == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Grader agent not found.")
			return
		}
		runs, err := gradingagentrepo.ListRunsByConfig(r.Context(), d.Pool, cfg.ID, 50)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load runs.")
			return
		}
		runJSON := make([]map[string]any, 0, len(runs))
		for _, run := range runs {
			entry := map[string]any{
				"id":             run.ID.String(),
				"scope":          string(run.Scope),
				"mode":           string(run.Mode),
				"status":         run.Status,
				"totalCount":     run.TotalCount,
				"completedCount": run.CompletedCount,
				"failedCount":    run.FailedCount,
				"createdAt":      run.CreatedAt.UTC().Format("2006-01-02T15:04:05.000000Z"),
			}
			if run.InitiatedBy != nil {
				entry["initiatedBy"] = run.InitiatedBy.String()
			}
			if run.FinishedAt != nil {
				entry["finishedAt"] = run.FinishedAt.UTC().Format("2006-01-02T15:04:05.000000Z")
			}
			if run.CancelledAt != nil {
				entry["cancelledAt"] = run.CancelledAt.UTC().Format("2006-01-02T15:04:05.000000Z")
			}
			if run.CancelledBy != nil {
				entry["cancelledBy"] = run.CancelledBy.String()
			}
			if run.ModelID != nil {
				entry["model"] = *run.ModelID
			} else if cfg.ModelID != nil {
				entry["model"] = *cfg.ModelID
			}
			if run.CostUSD != nil {
				entry["costUsd"] = *run.CostUSD
			}
			if run.PromptTokens != nil && *run.PromptTokens > 0 {
				entry["promptTokens"] = *run.PromptTokens
			}
			if run.CompletionTokens != nil && *run.CompletionTokens > 0 {
				entry["completionTokens"] = *run.CompletionTokens
			}
			if run.BudgetUSD != nil {
				entry["budgetUsd"] = *run.BudgetUSD
			}
			if filterJSON := runFilterToJSONFromBytes(run.Filter); filterJSON != nil {
				entry["filter"] = filterJSON
			}
			runJSON = append(runJSON, entry)
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{"runs": runJSON})
	}
}

func (d Deps) handleGetGraderAgentReviewQueue() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		courseCode, _, ok := d.requireGraderAgentReviewInboxAccess(w, r)
		if !ok {
			return
		}
		itemID, err := uuid.Parse(chi.URLParam(r, "item_id"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid assignment id.")
			return
		}
		cid, assignRow, ok := d.loadAssignmentForSubmissions(w, r, courseCode, itemID)
		if !ok || assignRow == nil {
			return
		}
		cfg, err := gradingagentrepo.GetConfigByItem(r.Context(), d.Pool, itemID)
		if err != nil || cfg == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Grader agent not found.")
			return
		}
		held, flagged, err := gradingagentrepo.ListReviewQueueByConfig(r.Context(), d.Pool, cfg.ID, 100)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load review queue.")
			return
		}
		submissionIDs := make([]uuid.UUID, 0, len(held)+len(flagged))
		for _, item := range held {
			submissionIDs = append(submissionIDs, item.SubmissionID)
		}
		for _, item := range flagged {
			submissionIDs = append(submissionIDs, item.SubmissionID)
		}
		labels, err := d.submissionLabelsForGraderAgentReview(r.Context(), *cid, itemID, assignRow, submissionIDs)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to resolve submission labels.")
			return
		}
		heldJSON := make([]map[string]any, 0, len(held))
		for _, item := range held {
			heldJSON = append(heldJSON, graderAgentReviewQueueItemToJSON(item, labels[item.SubmissionID]))
		}
		flaggedJSON := make([]map[string]any, 0, len(flagged))
		for _, item := range flagged {
			flaggedJSON = append(flaggedJSON, graderAgentReviewQueueItemToJSON(item, labels[item.SubmissionID]))
		}
		totalCount, _ := gradingagentrepo.CountReviewQueueByConfig(r.Context(), d.Pool, cfg.ID)
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"held":       heldJSON,
			"flagged":    flaggedJSON,
			"totalCount": totalCount,
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

