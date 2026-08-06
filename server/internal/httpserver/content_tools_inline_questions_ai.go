package httpserver

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"unicode/utf8"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/lextures/lextures/server/internal/apierr"
	ctrepo "github.com/lextures/lextures/server/internal/repos/contenttools"
	"github.com/lextures/lextures/server/internal/repos/course"
	"github.com/lextures/lextures/server/internal/repos/coursemoduleassignments"
	"github.com/lextures/lextures/server/internal/repos/coursemodulecontent"
	"github.com/lextures/lextures/server/internal/repos/systemprompts"
	"github.com/lextures/lextures/server/internal/repos/userai"
	"github.com/lextures/lextures/server/internal/service/aigateway"
	"github.com/lextures/lextures/server/internal/service/aiprovider"
	"github.com/lextures/lextures/server/internal/service/inlinequestionsai"
	"github.com/lextures/lextures/server/internal/service/outcomesextraction"
)

type buildInlineQuestionsWithAIRequest struct {
	PageMarkdown  *string `json:"pageMarkdown"`
	QuestionCount *int    `json:"questionCount"`
}

type buildInlineQuestionsWithAIResponse struct {
	Label     string `json:"label,omitempty"`
	Questions any    `json:"questions"`
}

func (d Deps) inlineQuestionsSystemPrompt(r *http.Request) string {
	if d.Pool == nil {
		return inlinequestionsai.DefaultSystemPrompt
	}
	if s, err := systemprompts.GetByKey(r.Context(), d.Pool, inlinequestionsai.PromptKey); err == nil && strings.TrimSpace(s) != "" {
		return s
	}
	return inlinequestionsai.DefaultSystemPrompt
}

// loadHostPageMaterial resolves title + markdown for a content-tool host when the client
// does not send draft pageMarkdown (saved body only).
func (d Deps) loadHostPageMaterial(
	r *http.Request,
	courseID uuid.UUID,
	courseCode string,
	inst *ctrepo.InstanceRow,
) (title, markdown string) {
	if inst == nil {
		return "", ""
	}
	switch inst.HostKind {
	case "assignment":
		if inst.StructureItemID == nil {
			return "", ""
		}
		row, err := coursemoduleassignments.GetForCourseItem(r.Context(), d.Pool, courseID, *inst.StructureItemID)
		if err != nil || row == nil {
			return "", ""
		}
		return row.Title, row.Markdown
	case "content_page":
		if inst.StructureItemID == nil {
			return "", ""
		}
		row, err := coursemodulecontent.GetForCourseItem(r.Context(), d.Pool, courseID, *inst.StructureItemID)
		if err != nil || row == nil {
			return "", ""
		}
		return row.Title, row.Markdown
	case "syllabus":
		p, err := course.GetSyllabusByCourseCode(r.Context(), d.Pool, courseCode)
		if err != nil || p == nil {
			return "Syllabus", ""
		}
		return "Syllabus", outcomesextraction.SyllabusPromptMaterial(p.Sections)
	default:
		return "", ""
	}
}

// handleBuildInlineQuestionsWithAI is POST
// /api/v1/courses/{course_code}/content-tools/instances/{instance_id}/build-with-ai
// Returns draft questions only; does not persist config.
func (d Deps) handleBuildInlineQuestionsWithAI() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		courseCode, viewer, courseID, ok := d.requireContentToolsCourse(w, r)
		if !ok {
			return
		}
		canEdit, err := d.viewerCanEditContentTools(r.Context(), courseCode, viewer)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to verify permissions.")
			return
		}
		if !canEdit {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "You do not have permission for this action.")
			return
		}

		instanceID, err := uuid.Parse(chi.URLParam(r, "instance_id"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid instance id.")
			return
		}
		inst, err := ctrepo.GetInstance(r.Context(), d.Pool, courseID, instanceID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load instance.")
			return
		}
		if inst == nil || inst.Status != "active" {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Instance not found.")
			return
		}
		if inst.ToolID != "inline_questions" {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Build with AI is only available for Inline Questions.")
			return
		}

		var body buildInlineQuestionsWithAIRequest
		if r.Body != nil {
			dec := json.NewDecoder(r.Body)
			if err := dec.Decode(&body); err != nil && !errors.Is(err, io.EOF) {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
				return
			}
		}

		hostTitle, hostMarkdown := d.loadHostPageMaterial(r, courseID, courseCode, inst)
		pageMarkdown := strings.TrimSpace(hostMarkdown)
		if body.PageMarkdown != nil && strings.TrimSpace(*body.PageMarkdown) != "" {
			pageMarkdown = strings.TrimSpace(*body.PageMarkdown)
		}
		if pageMarkdown == "" {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Add page or assignment content before building questions with AI.")
			return
		}
		if utf8.RuneCountInString(pageMarkdown) > inlinequestionsai.MaxPageMarkdownRunes {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Page content is too long.")
			return
		}

		questionCount := 2
		if body.QuestionCount != nil {
			questionCount = *body.QuestionCount
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
		if !d.enforceAIGateway(w, r, viewer, aigateway.FeatureInlineQuestionsGeneration, model, pageMarkdown) {
			return
		}
		gwDec := aigateway.Decision{
			UserIDHash:     aigateway.UserIDHash(d.aiGatewayConfig().HMACSecret, viewer),
			OptInConfirmed: true,
		}

		bound := aiprovider.BoundCompleter{Resolver: d.aiProviderResolver(), OrgID: orgID}
		result, callMeta, err := inlinequestionsai.Generate(
			r.Context(),
			bound,
			model,
			d.inlineQuestionsSystemPrompt(r),
			inlinequestionsai.GenerateInput{
				PageTitle:     hostTitle,
				PageMarkdown:  pageMarkdown,
				QuestionCount: questionCount,
			},
		)
		if err != nil {
			msg := err.Error()
			if strings.Contains(msg, "required") || strings.Contains(msg, "too long") {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, msg)
				return
			}
			if strings.Contains(msg, "parse inline questions JSON") || strings.Contains(msg, "could not find JSON") {
				writeAIGenerationFailed(w, r, "AI did not return valid inline questions JSON: "+msg, err)
				return
			}
			writeAIGenerationFailed(w, r, "AI generation failed: "+msg, err)
			return
		}
		if result == nil || len(result.Questions) == 0 {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "No questions could be generated from this content. Try adding more detail to the page.")
			return
		}

		d.logAIInferenceAllowedWithProvider(r, viewer, aigateway.FeatureInlineQuestionsGeneration, model, string(callMeta.Provider), pageMarkdown, gwDec)
		d.recordAIProviderUsage(r.Context(), AIUsageMeta{
			UserID: viewer, CourseID: &courseID, CourseCode: courseCode, Feature: aigateway.FeatureInlineQuestionsGeneration, Model: model,
		}, callMeta, true)

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(buildInlineQuestionsWithAIResponse{
			Label:     result.Label,
			Questions: result.Questions,
		})
	}
}
