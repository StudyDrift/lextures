package httpserver

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/lextures/lextures/server/internal/apierr"
	"github.com/lextures/lextures/server/internal/repos/systemprompts"
	userrepo "github.com/lextures/lextures/server/internal/repos/user"
	"github.com/lextures/lextures/server/internal/repos/userai"
	aigateway "github.com/lextures/lextures/server/internal/service/aigateway"
	"github.com/lextures/lextures/server/internal/service/notebookrag"
	"github.com/lextures/lextures/server/internal/service/openrouter"
)

type notebookRagJSON struct {
	Question  string `json:"question"`
	Notebooks []struct {
		CourseCode  string `json:"courseCode"`
		CourseTitle string `json:"courseTitle"`
		Markdown    string `json:"markdown"`
	} `json:"notebooks"`
}

func (d Deps) handleNotebookQuery() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		// No DB pool: match legacy behavior for misconfigured dev/test handlers (503 before auth).
		if d.Pool == nil && d.openRouterClient() == nil {
			apierr.WriteJSON(w, http.StatusServiceUnavailable, apierr.CodeAiNotConfigured, "AI features are not configured on this server.")
			return
		}
		var body notebookRagJSON
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		userID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		docs := make([]notebookrag.DocInput, 0, len(body.Notebooks))
		for _, n := range body.Notebooks {
			docs = append(docs, notebookrag.DocInput{
				CourseCode:  n.CourseCode,
				CourseTitle: n.CourseTitle,
				Markdown:    n.Markdown,
			})
		}
		docs = notebookrag.FilterDocs(docs)
		model := userai.DefaultCourseSetupModelID
		if m, err := userai.GetCourseSetupModelID(r.Context(), d.Pool, userID); err == nil && m != "" {
			model = m
		}
		contentKey := body.Question
		for _, doc := range docs {
			contentKey += doc.Markdown
		}
		if !d.enforceAIGateway(w, r, userID, aigateway.FeatureRAGNotebook, model, contentKey) {
			return
		}
		if d.openRouterClient() == nil {
			apierr.WriteJSON(w, http.StatusServiceUnavailable, apierr.CodeAiNotConfigured, "AI features are not configured on this server.")
			return
		}
		gwDec := aigateway.Decision{
			UserIDHash:     aigateway.UserIDHash(d.aiGatewayConfig().HMACSecret, userID),
			OptInConfirmed: true,
		}
		resp, err := notebookrag.Answer(r.Context(), d.Pool, d.openRouterClient(), userID, body.Question, docs)
		if err != nil {
			if notebookrag.IsValidationError(err) {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, err.Error())
				return
			}
			if notebookrag.IsGenerationError(err) {
				msg := err.Error()
				if len(msg) > 800 {
					msg = msg[:800]
				}
				apierr.WriteJSON(w, http.StatusBadGateway, apierr.CodeAiGenerationFailed, msg)
				return
			}
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not complete notebook query.")
			return
		}
		d.logAIInferenceAllowed(r, userID, aigateway.FeatureRAGNotebook, model, contentKey, gwDec)
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(resp)
	}
}

func (d Deps) handleGenerateNotebookFlashcards() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		if d.Pool == nil && d.openRouterClient() == nil {
			apierr.WriteJSON(w, http.StatusServiceUnavailable, apierr.CodeAiNotConfigured, "AI features are not configured on this server.")
			return
		}
		var body struct {
			Notes string `json:"notes"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		notes := strings.TrimSpace(body.Notes)
		if notes == "" {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Notes content cannot be empty.")
			return
		}

		userID, ok := d.meUserID(w, r)
		if !ok {
			return
		}

		model := userrepo.DefaultNotebookFlashcardsModelID
		if m, err := userrepo.GetNotebookFlashcardsModelID(r.Context(), d.Pool, userID); err == nil && m != "" {
			model = m
		}

		if !d.enforceAIGateway(w, r, userID, aigateway.FeatureRAGNotebook, model, notes) {
			return
		}

		if d.openRouterClient() == nil {
			apierr.WriteJSON(w, http.StatusServiceUnavailable, apierr.CodeAiNotConfigured, "AI features are not configured on this server.")
			return
		}

		// Load system prompt from database, or fallback
		sysPrompt, err := systemprompts.GetByKey(r.Context(), d.Pool, "notebook_flashcards")
		if err != nil {
			// Fallback to hardcoded default
			sysPrompt = systemprompts.DefaultNotebookFlashcardsPrompt
		}

		messages := []openrouter.Message{
			{Role: "system", Content: sysPrompt},
			{Role: "user", Content: notes},
		}

		gwDec := aigateway.Decision{
			UserIDHash:     aigateway.UserIDHash(d.aiGatewayConfig().HMACSecret, userID),
			OptInConfirmed: true,
		}

		completion, err := d.openRouterClient().ChatCompletion(model, messages)
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadGateway, apierr.CodeAiGenerationFailed, fmt.Sprintf("AI model failed to respond: %v", err))
			return
		}

		d.logAIInferenceAllowed(r, userID, aigateway.FeatureRAGNotebook, model, notes, gwDec)

		// Parse output to ensure it is valid JSON matching our expected flashcards structure
		var parsed struct {
			Flashcards []struct {
				Front string `json:"front"`
				Back  string `json:"back"`
			} `json:"flashcards"`
		}

		// Sometimes AI outputs Markdown JSON blocks (e.g. ```json ... ```)
		cleanCompletion := completion
		if idx := strings.Index(cleanCompletion, "```json"); idx != -1 {
			cleanCompletion = cleanCompletion[idx+7:]
			if endIdx := strings.Index(cleanCompletion, "```"); endIdx != -1 {
				cleanCompletion = cleanCompletion[:endIdx]
			}
		}
		cleanCompletion = strings.TrimSpace(cleanCompletion)

		if err := json.Unmarshal([]byte(cleanCompletion), &parsed); err != nil {
			apierr.WriteJSON(w, http.StatusBadGateway, apierr.CodeAiGenerationFailed, fmt.Sprintf("AI did not return valid JSON: %v. Raw response: %s", err, completion))
			return
		}

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(parsed)
	}
}

