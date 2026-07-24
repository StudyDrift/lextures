package httpserver

import (
	"encoding/json"
	"net/http"
	"strings"
	"unicode/utf8"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/lextures/lextures/server/internal/apierr"
	"github.com/lextures/lextures/server/internal/models/coursemodulecontent"
	"github.com/lextures/lextures/server/internal/repos/course"
	"github.com/lextures/lextures/server/internal/repos/coursemoduleassignments"
	"github.com/lextures/lextures/server/internal/repos/rbac"
	"github.com/lextures/lextures/server/internal/repos/systemprompts"
	"github.com/lextures/lextures/server/internal/repos/userai"
	"github.com/lextures/lextures/server/internal/service/aigateway"
	"github.com/lextures/lextures/server/internal/service/aiprovider"
	"github.com/lextures/lextures/server/internal/service/assignmentrubricai"
)

func (d Deps) requireAssignmentItemEdit(w http.ResponseWriter, r *http.Request) (
	courseCode string,
	courseID, itemID, viewer uuid.UUID,
	row *coursemoduleassignments.CourseItemAssignmentRow,
	ok bool,
) {
	courseCode, viewer, ok = d.requireCourseAccess(w, r)
	if !ok {
		return "", uuid.Nil, uuid.Nil, uuid.Nil, nil, false
	}
	parsedItem, err := uuid.Parse(chi.URLParam(r, "item_id"))
	if err != nil {
		apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid item id.")
		return "", uuid.Nil, uuid.Nil, uuid.Nil, nil, false
	}
	perm := "course:" + courseCode + ":item:create"
	canEdit, err := rbac.UserHasPermission(r.Context(), d.Pool, viewer, perm)
	if err != nil {
		apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to verify permissions.")
		return "", uuid.Nil, uuid.Nil, uuid.Nil, nil, false
	}
	if !canEdit {
		apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "You do not have permission for this action.")
		return "", uuid.Nil, uuid.Nil, uuid.Nil, nil, false
	}
	cid, err := course.GetIDByCourseCode(r.Context(), d.Pool, courseCode)
	if err != nil {
		apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load course.")
		return "", uuid.Nil, uuid.Nil, uuid.Nil, nil, false
	}
	if cid == nil {
		apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Course not found.")
		return "", uuid.Nil, uuid.Nil, uuid.Nil, nil, false
	}
	row, err = coursemoduleassignments.GetForCourseItem(r.Context(), d.Pool, *cid, parsedItem)
	if err != nil {
		apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load assignment.")
		return "", uuid.Nil, uuid.Nil, uuid.Nil, nil, false
	}
	if row == nil {
		apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Not found.")
		return "", uuid.Nil, uuid.Nil, uuid.Nil, nil, false
	}
	return courseCode, *cid, parsedItem, viewer, row, true
}

func (d Deps) assignmentRubricSystemPrompt(r *http.Request) string {
	if d.Pool == nil {
		return assignmentrubricai.DefaultSystemPrompt
	}
	if s, err := systemprompts.GetByKey(r.Context(), d.Pool, assignmentrubricai.PromptKey); err == nil && strings.TrimSpace(s) != "" {
		return s
	}
	return assignmentrubricai.DefaultSystemPrompt
}

// handleGenerateAssignmentRubric is POST .../assignments/{item_id}/generate-rubric
func (d Deps) handleGenerateAssignmentRubric() http.HandlerFunc {
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
		courseCode, courseID, _, viewer, row, ok := d.requireAssignmentItemEdit(w, r)
		if !ok {
			return
		}
		var body coursemodulecontent.GenerateAssignmentRubricRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		prompt := strings.TrimSpace(body.Prompt)
		if prompt == "" {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Instructions are required.")
			return
		}
		if utf8.RuneCountInString(prompt) > assignmentrubricai.MaxPromptRunes {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Instructions are too long.")
			return
		}
		var assignmentMarkdown string
		if body.AssignmentMarkdown != nil {
			assignmentMarkdown = *body.AssignmentMarkdown
			if utf8.RuneCountInString(assignmentMarkdown) > assignmentrubricai.MaxAssignmentMarkdownRunes {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Assignment body is too long.")
				return
			}
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
		promptMaterial := prompt + strings.TrimSpace(assignmentMarkdown)
		if !d.enforceAIGateway(w, r, viewer, aigateway.FeatureAssignmentRubricGeneration, model, promptMaterial) {
			return
		}
		gwDec := aigateway.Decision{UserIDHash: aigateway.UserIDHash(d.aiGatewayConfig().HMACSecret, viewer), OptInConfirmed: true}

		bound := aiprovider.BoundCompleter{Resolver: d.aiProviderResolver(), OrgID: orgID}
		rubric, callMeta, err := assignmentrubricai.Generate(r.Context(), bound, model, d.assignmentRubricSystemPrompt(r), assignmentrubricai.GenerateInput{
			Prompt:             prompt,
			AssignmentTitle:    row.Title,
			PointsWorth:        row.PointsWorth,
			AssignmentMarkdown: assignmentMarkdown,
		})
		if err != nil {
			msg := err.Error()
			if strings.Contains(msg, "too long") || strings.Contains(msg, "required") {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, msg)
				return
			}
			writeAIGenerationFailed(w, r, "AI generation failed: "+msg, err)
			return
		}

		d.logAIInferenceAllowedWithProvider(r, viewer, aigateway.FeatureAssignmentRubricGeneration, model, string(callMeta.Provider), promptMaterial, gwDec)
		d.recordAIProviderUsage(r.Context(), AIUsageMeta{
			UserID: viewer, CourseID: &courseID, CourseCode: courseCode, Feature: aigateway.FeatureAssignmentRubricGeneration, Model: model,
		}, callMeta, true)

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(coursemodulecontent.GenerateAssignmentRubricResponse{Rubric: *rubric})
	}
}
