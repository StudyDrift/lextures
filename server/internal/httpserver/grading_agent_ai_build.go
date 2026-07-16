package httpserver

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/lextures/lextures/server/internal/apierr"
	gradingagentrepo "github.com/lextures/lextures/server/internal/repos/gradingagent"
	"github.com/lextures/lextures/server/internal/service/aigateway"
	gradingagentsvc "github.com/lextures/lextures/server/internal/service/gradingagent"
)

// graderAgentAiBuilderEnabled gates the natural-language workflow builder. It
// reuses the grader-agent enablement and requires an AI provider to be wired,
// avoiding a dedicated platform flag/migration.
func (d Deps) graderAgentAiBuilderEnabled() bool {
	return d.graderAgentEnabled() && d.aiConfigured(context.Background(), nil)
}

type graderAgentAIBuildBody struct {
	Instruction  string                         `json:"instruction"`
	CurrentGraph *gradingagentsvc.WorkflowGraph `json:"currentGraph"`
	QuizSlots    []graderAgentAIBuildQuizSlot   `json:"quizSlots"`
	MaxPoints    float64                        `json:"maxPoints"`
}

type graderAgentAIBuildQuizSlot struct {
	Index        int     `json:"index"`
	Label        string  `json:"label"`
	QuestionType string  `json:"questionType"`
	MaxPoints    float64 `json:"maxPoints"`
}

// handlePostGraderAgentAIBuild generates (or modifies) a grading-agent workflow
// graph from a plain-English instruction using the platform's registered AI.
// The graph is returned for in-canvas review; it is NOT persisted here.
func (d Deps) handlePostGraderAgentAIBuild() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.graderAgentAiBuilderEnabled() {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "AI workflow builder is not enabled.")
			return
		}
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

		payload, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Could not read body.")
			return
		}
		var body graderAgentAIBuildBody
		if err := json.Unmarshal(payload, &body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		instruction := strings.TrimSpace(body.Instruction)
		if instruction == "" {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Describe what the grading agent should do.")
			return
		}

		var configModelID *string
		if cfg, cfgErr := gradingagentrepo.GetConfigByItem(r.Context(), d.Pool, itemID); cfgErr == nil && cfg != nil {
			configModelID = cfg.ModelID
		}
		modelID, modelErr := d.resolveGraderAgentModelID(r.Context(), viewer, "", configModelID)
		if modelErr != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, gradingagentsvc.UserFacingScoreError(modelErr))
			return
		}

		dec, _ := aigateway.Evaluate(
			r.Context(), d.Pool, d.aiGatewayConfig(), viewer, nil,
			aigateway.FeatureGraderAgent, modelID,
			aigateway.ContentHash(instruction),
		)
		if !dec.Allowed {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, aigateway.BlockMessage(dec.Reason))
			return
		}

		orgID := d.orgIDPtrForUser(r.Context(), viewer)
		if !d.aiConfigured(r.Context(), orgID) {
			apierr.WriteJSON(w, http.StatusServiceUnavailable, apierr.CodeAiNotConfigured, aiNotConfiguredMsg)
			return
		}
		svc := d.gradingAgentService(orgID)
		if svc.AI == nil {
			apierr.WriteJSON(w, http.StatusServiceUnavailable, apierr.CodeAiNotConfigured, aiNotConfiguredMsg)
			return
		}

		isQuiz := item.Kind == "quiz"
		systemPrompt := gradingagentsvc.BuildWorkflowBuilderSystemPrompt(gradingagentsvc.BuilderPromptOptions{
			IsQuiz:    isQuiz,
			QuizSlots: builderQuizSlots(body.QuizSlots),
			MaxPoints: body.MaxPoints,
		})

		result, genErr := d.generateGraderAgentGraph(r.Context(), svc, modelID, systemPrompt, instruction, body.CurrentGraph)
		if genErr != nil {
			if ve, isVE := genErr.(gradingagentsvc.ValidationError); isVE {
				apierr.WriteJSON(w, http.StatusUnprocessableEntity, apierr.CodeInvalidInput,
					"The AI produced an invalid workflow: "+ve.Message+" Try rephrasing your instruction.")
				return
			}
			if isTimeoutError(genErr) {
				apierr.WriteJSON(w, http.StatusGatewayTimeout, apierr.CodeInternal,
					"The AI model took too long to respond. Try again, simplify the instruction, or select a faster grading model in Settings.")
				return
			}
			apierr.WriteJSON(w, http.StatusBadGateway, apierr.CodeInternal, "AI workflow generation failed: "+genErr.Error())
			return
		}

		graphJSON, marshalErr := gradingagentsvc.WorkflowGraphToJSON(result.Graph)
		if marshalErr != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not serialize generated workflow.")
			return
		}

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"workflowGraph": json.RawMessage(graphJSON),
			"summary":       result.Summary,
		})
	}
}

func isTimeoutError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "deadline exceeded") ||
		strings.Contains(msg, "Client.Timeout") ||
		strings.Contains(msg, "context canceled") ||
		strings.Contains(msg, "timeout")
}

func builderQuizSlots(in []graderAgentAIBuildQuizSlot) []gradingagentsvc.BuilderQuizSlot {
	if len(in) == 0 {
		return nil
	}
	out := make([]gradingagentsvc.BuilderQuizSlot, 0, len(in))
	for _, s := range in {
		out = append(out, gradingagentsvc.BuilderQuizSlot{
			Index:        s.Index,
			Label:        s.Label,
			QuestionType: s.QuestionType,
			MaxPoints:    s.MaxPoints,
		})
	}
	return out
}

// graderAgentBuildTimeout bounds each model call for workflow generation. Longer
// than the shared 120s client timeout because a single structured-graph generation
// can be slower than a grading call.
const graderAgentBuildTimeout = 180 * time.Second

// graderAgentBuildMaxTokens caps generation length; a workflow graph fits well
// within this, and the cap stops a model from running past the timeout.
const graderAgentBuildMaxTokens = 8000

// generateGraderAgentGraph runs the model, validates the result, and retries once
// with the validation error fed back if the first attempt is parseable but invalid.
func (d Deps) generateGraderAgentGraph(
	ctx context.Context,
	svc *gradingagentsvc.Service,
	modelID, systemPrompt, instruction string,
	currentGraph *gradingagentsvc.WorkflowGraph,
) (gradingagentsvc.BuilderResult, error) {
	// Use a dedicated timeout for this single-shot generation, longer than the
	// shared client timeout because a structured-graph generation can be slower.
	buildCtx, cancel := context.WithTimeout(ctx, graderAgentBuildTimeout)
	defer cancel()

	input := ""
	if currentGraph != nil && len(currentGraph.Nodes) > 0 {
		if raw, err := gradingagentsvc.WorkflowGraphToJSON(currentGraph); err == nil {
			input = "Current graph to modify:\n" + string(raw)
		}
	}

	var lastErr error
	for attempt := 0; attempt < 2; attempt++ {
		text, _, _, _, runErr := svc.RunBuilderPrompt(buildCtx, modelID, systemPrompt, instruction, input, graderAgentBuildMaxTokens)
		if runErr != nil {
			// A timeout or transport error won't be helped by retrying with the
			// same slow model; surface it immediately.
			return gradingagentsvc.BuilderResult{}, runErr
		}
		result, parseErr := gradingagentsvc.ParseBuilderResponse(text)
		if parseErr == nil {
			if valErr := gradingagentsvc.ValidateWorkflowGraphForPersistence(result.Graph); valErr == nil {
				return result, nil
			} else {
				lastErr = valErr
			}
		} else {
			lastErr = parseErr
		}
		// Feed the failure back for one repair attempt.
		input = fmt.Sprintf("Your previous response was rejected: %s\nReturn corrected JSON in the required envelope.\nYour previous response was:\n%s", lastErr.Error(), text)
	}
	return gradingagentsvc.BuilderResult{}, lastErr
}
