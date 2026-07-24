package httpserver

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/lextures/lextures/server/internal/apierr"
	"github.com/lextures/lextures/server/internal/repos/course"
	"github.com/lextures/lextures/server/internal/repos/coursemodulecontent"
	"github.com/lextures/lextures/server/internal/repos/coursemodulequizzes"
	"github.com/lextures/lextures/server/internal/repos/rbac"
	"github.com/lextures/lextures/server/internal/repos/userai"
	"github.com/lextures/lextures/server/internal/service/aigateway"
	"github.com/lextures/lextures/server/internal/service/aiprovider"
	"github.com/lextures/lextures/server/internal/service/contentpagegeneration"
)

type buildPageBodyWithAIRequest struct {
	Prompt           string `json:"prompt"`
	ExistingMarkdown string `json:"existingMarkdown"`
}

type buildPageBodyWithAIResponse struct {
	Sections []contentpagegeneration.DraftSection `json:"sections"`
}

func (d Deps) writeBuildPageBodyWithAI(
	w http.ResponseWriter,
	r *http.Request,
	courseCode string,
	courseID, viewer uuid.UUID,
	pageTitle, prompt, existingMarkdown string,
) {
	prompt = strings.TrimSpace(prompt)
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

	promptMaterial := prompt + strings.TrimSpace(existingMarkdown)
	if !d.enforceAIGateway(w, r, viewer, aigateway.FeatureContentPageGeneration, model, promptMaterial) {
		return
	}
	gwDec := aigateway.Decision{
		UserIDHash:     aigateway.UserIDHash(d.aiGatewayConfig().HMACSecret, viewer),
		OptInConfirmed: true,
	}

	bound := aiprovider.BoundCompleter{Resolver: d.aiProviderResolver(), OrgID: orgID}
	sections, callMeta, err := contentpagegeneration.GenerateFromPrompt(
		r.Context(),
		bound,
		model,
		contentpagegeneration.DefaultSystemPrompt,
		prompt,
		existingMarkdown,
		pageTitle,
	)
	if err != nil {
		msg := err.Error()
		if strings.Contains(msg, "parse content page sections JSON") {
			writeAIGenerationFailed(w, r, "AI did not return valid content page JSON: "+msg, err)
			return
		}
		if strings.Contains(msg, "too long") {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, msg)
			return
		}
		writeAIGenerationFailed(w, r, "AI generation failed: "+msg, err)
		return
	}

	d.logAIInferenceAllowedWithProvider(r, viewer, aigateway.FeatureContentPageGeneration, model, string(callMeta.Provider), promptMaterial, gwDec)
	d.recordAIProviderUsage(r.Context(), AIUsageMeta{
		UserID: viewer, CourseID: &courseID, CourseCode: courseCode, Feature: aigateway.FeatureContentPageGeneration, Model: model,
	}, callMeta, true)

	if sections == nil {
		sections = []contentpagegeneration.DraftSection{}
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(buildPageBodyWithAIResponse{Sections: sections})
}

func (d Deps) requireContentPageEdit(
	w http.ResponseWriter,
	r *http.Request,
) (courseCode string, courseID, itemID, viewer uuid.UUID, pageTitle string, ok bool) {
	courseCode, viewer, ok = d.requireCourseAccess(w, r)
	if !ok {
		return "", uuid.Nil, uuid.Nil, uuid.Nil, "", false
	}
	parsedItem, err := uuid.Parse(chi.URLParam(r, "item_id"))
	if err != nil {
		apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid item id.")
		return "", uuid.Nil, uuid.Nil, uuid.Nil, "", false
	}
	perm := "course:" + courseCode + ":item:create"
	canEdit, err := rbac.UserHasPermission(r.Context(), d.Pool, viewer, perm)
	if err != nil {
		apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to verify permissions.")
		return "", uuid.Nil, uuid.Nil, uuid.Nil, "", false
	}
	if !canEdit {
		apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "You do not have permission for this action.")
		return "", uuid.Nil, uuid.Nil, uuid.Nil, "", false
	}
	cid, err := course.GetIDByCourseCode(r.Context(), d.Pool, courseCode)
	if err != nil {
		apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load course.")
		return "", uuid.Nil, uuid.Nil, uuid.Nil, "", false
	}
	if cid == nil {
		apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Course not found.")
		return "", uuid.Nil, uuid.Nil, uuid.Nil, "", false
	}
	row, err := coursemodulecontent.GetForCourseItem(r.Context(), d.Pool, *cid, parsedItem)
	if err != nil {
		apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load content page.")
		return "", uuid.Nil, uuid.Nil, uuid.Nil, "", false
	}
	if row == nil {
		apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Not found.")
		return "", uuid.Nil, uuid.Nil, uuid.Nil, "", false
	}
	return courseCode, *cid, parsedItem, viewer, row.Title, true
}

// handleBuildModuleContentPageWithAI is POST /api/v1/courses/{course_code}/content-pages/{item_id}/build-with-ai.
// Returns draft sections only; does not persist.
func (d Deps) handleBuildModuleContentPageWithAI() http.HandlerFunc {
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
		courseCode, courseID, _, viewer, pageTitle, ok := d.requireContentPageEdit(w, r)
		if !ok {
			return
		}
		var body buildPageBodyWithAIRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		d.writeBuildPageBodyWithAI(w, r, courseCode, courseID, viewer, pageTitle, body.Prompt, body.ExistingMarkdown)
	}
}

// handleBuildModuleQuizIntroWithAI is POST /api/v1/courses/{course_code}/quizzes/{item_id}/build-intro-with-ai.
// Returns draft sections only; does not persist.
func (d Deps) handleBuildModuleQuizIntroWithAI() http.HandlerFunc {
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
		courseCode, courseID, itemID, viewer, ok := d.requireQuizItemEdit(w, r)
		if !ok {
			return
		}
		row, err := coursemodulequizzes.GetForCourseItem(r.Context(), d.Pool, courseID, itemID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load quiz.")
			return
		}
		if row == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Not found.")
			return
		}
		var body buildPageBodyWithAIRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		d.writeBuildPageBodyWithAI(w, r, courseCode, courseID, viewer, row.Title, body.Prompt, body.ExistingMarkdown)
	}
}
