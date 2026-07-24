package httpserver

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/lextures/lextures/server/internal/apierr"
	"github.com/lextures/lextures/server/internal/repos/course"
	"github.com/lextures/lextures/server/internal/repos/enrollment"
	userai "github.com/lextures/lextures/server/internal/repos/user"
	aigateway "github.com/lextures/lextures/server/internal/service/aigateway"
	"github.com/lextures/lextures/server/internal/service/aiprovider"
	"github.com/lextures/lextures/server/internal/service/outcomesextraction"
)

// handleExtractCourseOutcomesFromSyllabus is POST /api/v1/courses/{course_code}/outcomes/extract-from-syllabus.
// Returns draft outcomes only; does not persist.
func (d Deps) handleExtractCourseOutcomesFromSyllabus() http.HandlerFunc {
	type resp struct {
		Outcomes []outcomesextraction.DraftOutcome `json:"outcomes"`
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
		courseCode, viewer, ok := d.requireCourseAccess(w, r)
		if !ok {
			return
		}
		isStaff, err := enrollment.UserIsCourseStaff(r.Context(), d.Pool, courseCode, viewer)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to verify access.")
			return
		}
		if !isStaff {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "Forbidden.")
			return
		}

		p, err := course.GetSyllabusByCourseCode(r.Context(), d.Pool, courseCode)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load syllabus.")
			return
		}
		if p == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Course not found.")
			return
		}
		syllabusMarkdown := outcomesextraction.SyllabusPromptMaterial(p.Sections)
		if syllabusMarkdown == "" {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Add syllabus content before extracting outcomes.")
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

		if !d.enforceAIGateway(w, r, viewer, aigateway.FeatureOutcomesExtraction, model, syllabusMarkdown) {
			return
		}
		gwDec := aigateway.Decision{
			UserIDHash:     aigateway.UserIDHash(d.aiGatewayConfig().HMACSecret, viewer),
			OptInConfirmed: true,
		}

		bound := aiprovider.BoundCompleter{Resolver: d.aiProviderResolver(), OrgID: orgID}
		outcomes, callMeta, err := outcomesextraction.ExtractFromSyllabus(
			r.Context(), bound, model, outcomesextraction.DefaultSystemPrompt, syllabusMarkdown,
		)
		if err != nil {
			msg := err.Error()
			if strings.Contains(msg, "parse outcomes JSON") {
				writeAIGenerationFailed(w, r, "AI did not return valid outcomes JSON: "+msg, err)
				return
			}
			if strings.Contains(msg, "too long") {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, msg)
				return
			}
			writeAIGenerationFailed(w, r, "AI generation failed: "+msg, err)
			return
		}

		d.logAIInferenceAllowedWithProvider(r, viewer, aigateway.FeatureOutcomesExtraction, model, string(callMeta.Provider), syllabusMarkdown, gwDec)
		d.recordAIProviderUsage(r.Context(), AIUsageMeta{
			UserID: viewer, CourseCode: courseCode, Feature: aigateway.FeatureOutcomesExtraction, Model: model,
		}, callMeta, true)

		if outcomes == nil {
			outcomes = []outcomesextraction.DraftOutcome{}
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(resp{Outcomes: outcomes})
	}
}
