package httpserver

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/lextures/lextures/server/internal/apierr"
	"github.com/lextures/lextures/server/internal/repos/user"
	"github.com/lextures/lextures/server/internal/service/openrouter"
)

// handleListAIModels is GET /api/v1/settings/ai/models?kind=text|image
func (d Deps) handleListAIModels() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet+","+http.MethodOptions)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		if _, ok := d.adminRbacUser(w, r); !ok {
			return
		}
		kind := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("kind")))
		if kind == "" {
			kind = "text"
		}
		if kind != "text" && kind != "image" {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid kind (use text or image).")
			return
		}
		models, err := openrouter.ListModelsByOutputModality(r.Context(), nil, openrouter.DefaultBaseURL, kind)
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput,
				"Could not load models from OpenRouter. Try again. ("+err.Error()+")")
			return
		}
		configured := d.openRouterClient() != nil
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"configured": configured,
			"models":     models,
		})
	}
}

// handleGetSettingsAI is GET /api/v1/settings/ai
func (d Deps) handleGetSettingsAI() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet+","+http.MethodOptions)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		uid, ok := d.adminRbacUser(w, r)
		if !ok {
			return
		}
		img, err := user.GetImageModelID(r.Context(), d.Pool, uid)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load AI settings.")
			return
		}
		course, err := user.GetCourseSetupModelID(r.Context(), d.Pool, uid)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load AI settings.")
			return
		}
		flashcards, err := user.GetNotebookFlashcardsModelID(r.Context(), d.Pool, uid)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load AI settings.")
			return
		}
		vibe, err := user.GetVibeActivityModelID(r.Context(), d.Pool, uid)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load AI settings.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]string{
			"imageModelId":                img,
			"courseSetupModelId":          course,
			"notebookFlashcardsModelId":   flashcards,
			"vibeActivityModelId":         vibe,
		})
	}
}

type putSettingsAIBody struct {
	ImageModelID                string `json:"imageModelId"`
	CourseSetupModelID          string `json:"courseSetupModelId"`
	NotebookFlashcardsModelID   string `json:"notebookFlashcardsModelId"`
	VibeActivityModelID         string `json:"vibeActivityModelId"`
}

// handlePutSettingsAI is PUT /api/v1/settings/ai
func (d Deps) handlePutSettingsAI() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if r.Method != http.MethodPut {
			w.Header().Set("Allow", http.MethodPut+","+http.MethodOptions)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		uid, ok := d.adminRbacUser(w, r)
		if !ok {
			return
		}
		b, _ := io.ReadAll(r.Body)
		_ = r.Body.Close()
		var in putSettingsAIBody
		if err := json.Unmarshal(b, &in); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		img := strings.TrimSpace(in.ImageModelID)
		if img == "" {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Choose an image model.")
			return
		}
		course := strings.TrimSpace(in.CourseSetupModelID)
		if course == "" {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Choose a course setup model.")
			return
		}
		flashcards := strings.TrimSpace(in.NotebookFlashcardsModelID)
		if flashcards == "" {
			flashcards = user.DefaultNotebookFlashcardsModelID
		}
		vibe := strings.TrimSpace(in.VibeActivityModelID)
		if vibe == "" {
			vibe = user.DefaultVibeActivityModelID
		}
		imgOut, courseOut, flashcardsOut, vibeOut, err := user.UpsertAISettings(r.Context(), d.Pool, uid, img, course, flashcards, vibe)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to save AI settings.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]string{
			"imageModelId":                imgOut,
			"courseSetupModelId":          courseOut,
			"notebookFlashcardsModelId":   flashcardsOut,
			"vibeActivityModelId":         vibeOut,
		})
	}
}
