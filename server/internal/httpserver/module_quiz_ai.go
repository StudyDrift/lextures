package httpserver

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/lextures/lextures/server/internal/apierr"
	"github.com/lextures/lextures/server/internal/models/coursemodulequiz"
	"github.com/lextures/lextures/server/internal/repos/course"
	"github.com/lextures/lextures/server/internal/repos/coursemodulequizzes"
	"github.com/lextures/lextures/server/internal/repos/rbac"
	"github.com/lextures/lextures/server/internal/repos/systemprompts"
	"github.com/lextures/lextures/server/internal/repos/userai"
	"github.com/lextures/lextures/server/internal/service/aigateway"
	"github.com/lextures/lextures/server/internal/service/aiprovider"
	"github.com/lextures/lextures/server/internal/service/quizgenerationai"
)

func (d Deps) requireQuizItemEdit(w http.ResponseWriter, r *http.Request) (courseCode string, courseID, itemID, viewer uuid.UUID, ok bool) {
	courseCode, viewer, ok = d.requireCourseAccess(w, r)
	if !ok {
		return "", uuid.Nil, uuid.Nil, uuid.Nil, false
	}
	parsedItem, err := uuid.Parse(chi.URLParam(r, "item_id"))
	if err != nil {
		apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid item id.")
		return "", uuid.Nil, uuid.Nil, uuid.Nil, false
	}
	perm := "course:" + courseCode + ":item:create"
	canEdit, err := rbac.UserHasPermission(r.Context(), d.Pool, viewer, perm)
	if err != nil {
		apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to verify permissions.")
		return "", uuid.Nil, uuid.Nil, uuid.Nil, false
	}
	if !canEdit {
		apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "You do not have permission for this action.")
		return "", uuid.Nil, uuid.Nil, uuid.Nil, false
	}
	cid, err := course.GetIDByCourseCode(r.Context(), d.Pool, courseCode)
	if err != nil {
		apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load course.")
		return "", uuid.Nil, uuid.Nil, uuid.Nil, false
	}
	if cid == nil {
		apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Course not found.")
		return "", uuid.Nil, uuid.Nil, uuid.Nil, false
	}
	row, err := coursemodulequizzes.GetForCourseItem(r.Context(), d.Pool, *cid, parsedItem)
	if err != nil {
		apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load quiz.")
		return "", uuid.Nil, uuid.Nil, uuid.Nil, false
	}
	if row == nil {
		apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Not found.")
		return "", uuid.Nil, uuid.Nil, uuid.Nil, false
	}
	return courseCode, *cid, parsedItem, viewer, true
}

func (d Deps) quizGenerationSystemPrompt(r *http.Request, key, fallback string) string {
	if d.Pool == nil {
		return fallback
	}
	if s, err := systemprompts.GetByKey(r.Context(), d.Pool, key); err == nil && strings.TrimSpace(s) != "" {
		return s
	}
	return fallback
}

// handleGenerateModuleQuizQuestions is POST .../quizzes/{item_id}/generate-questions
func (d Deps) handleGenerateModuleQuizQuestions() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost+","+http.MethodOptions)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		courseCode, courseID, _, viewer, ok := d.requireQuizItemEdit(w, r)
		if !ok {
			return
		}
		var body coursemodulequiz.GenerateModuleQuizQuestionsRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		prompt := strings.TrimSpace(body.Prompt)
		if prompt == "" {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Prompt is required.")
			return
		}
		orgID := d.orgIDPtrForUser(r.Context(), viewer)
		if !d.aiConfigured(r.Context(), orgID) {
			apierr.WriteJSON(w, http.StatusServiceUnavailable, apierr.CodeAiNotConfigured, aiNotConfiguredMsg)
			return
		}
		model, err := userai.GetCourseSetupModelID(r.Context(), d.Pool, viewer)
		if err != nil {
			model = userai.DefaultCourseSetupModelID
		}
		if !d.enforceAIGateway(w, r, viewer, aigateway.FeatureQuizGeneration, model, prompt) {
			return
		}
		gwDec := aigateway.Decision{UserIDHash: aigateway.UserIDHash(d.aiGatewayConfig().HMACSecret, viewer), OptInConfirmed: true}
		sys := d.quizGenerationSystemPrompt(r, "quiz_generation", quizgenerationai.DefaultSystemPrompt)
		bound := aiprovider.BoundCompleter{Resolver: d.aiProviderResolver(), OrgID: orgID}
		questions, callMeta, err := quizgenerationai.GenerateFromPrompt(r.Context(), bound, model, sys, prompt, int(body.QuestionCount))
		if err != nil {
			writeAIGenerationFailed(w, r, "AI generation failed: "+err.Error(), err)
			return
		}
		d.logAIInferenceAllowedWithProvider(r, viewer, aigateway.FeatureQuizGeneration, model, string(callMeta.Provider), prompt, gwDec)
		d.recordAIProviderUsage(r.Context(), AIUsageMeta{
			UserID: viewer, CourseID: &courseID, CourseCode: courseCode, Feature: aigateway.FeatureQuizGeneration, Model: model,
		}, callMeta, true)
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(coursemodulequiz.GenerateModuleQuizQuestionsResponse{Questions: questions})
	}
}

// handleImportModuleQuizQuestionsMarkdown is POST .../quizzes/{item_id}/import-questions-markdown
func (d Deps) handleImportModuleQuizQuestionsMarkdown() http.HandlerFunc {
	type reqBody struct {
		Markdown string `json:"markdown"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost+","+http.MethodOptions)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		courseCode, courseID, _, viewer, ok := d.requireQuizItemEdit(w, r)
		if !ok {
			return
		}
		var body reqBody
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		markdown := strings.TrimSpace(body.Markdown)
		if markdown == "" {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Markdown is required.")
			return
		}
		orgID := d.orgIDPtrForUser(r.Context(), viewer)
		if !d.aiConfigured(r.Context(), orgID) {
			apierr.WriteJSON(w, http.StatusServiceUnavailable, apierr.CodeAiNotConfigured, aiNotConfiguredMsg)
			return
		}
		model, err := userai.GetCourseSetupModelID(r.Context(), d.Pool, viewer)
		if err != nil {
			model = userai.DefaultCourseSetupModelID
		}
		if !d.enforceAIGateway(w, r, viewer, aigateway.FeatureQuizGeneration, model, markdown) {
			return
		}
		gwDec := aigateway.Decision{UserIDHash: aigateway.UserIDHash(d.aiGatewayConfig().HMACSecret, viewer), OptInConfirmed: true}
		sys := d.quizGenerationSystemPrompt(r, "quiz_markdown_import", quizgenerationai.DefaultMarkdownImportSystemPrompt)
		bound := aiprovider.BoundCompleter{Resolver: d.aiProviderResolver(), OrgID: orgID}
		questions, callMeta, err := quizgenerationai.ParseMarkdown(r.Context(), bound, model, sys, markdown)
		if err != nil {
			msg := err.Error()
			if strings.Contains(msg, "required") || strings.Contains(msg, "too long") {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, msg)
				return
			}
			writeAIGenerationFailed(w, r, "AI import failed: "+msg, err)
			return
		}
		if len(questions) == 0 {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "No questions could be parsed from the markdown.")
			return
		}
		d.logAIInferenceAllowedWithProvider(r, viewer, aigateway.FeatureQuizGeneration, model, string(callMeta.Provider), markdown, gwDec)
		d.recordAIProviderUsage(r.Context(), AIUsageMeta{
			UserID: viewer, CourseID: &courseID, CourseCode: courseCode, Feature: aigateway.FeatureQuizGeneration, Model: model,
		}, callMeta, true)
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(coursemodulequiz.GenerateModuleQuizQuestionsResponse{Questions: questions})
	}
}
