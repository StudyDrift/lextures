package httpserver

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/lextures/lextures/server/internal/apierr"
	badgerepo "github.com/lextures/lextures/server/internal/repos/badges"
	"github.com/lextures/lextures/server/internal/repos/course"
	"github.com/lextures/lextures/server/internal/repos/courseoutcomes"
	userai "github.com/lextures/lextures/server/internal/repos/user"
	aigateway "github.com/lextures/lextures/server/internal/service/aigateway"
	"github.com/lextures/lextures/server/internal/service/aiprovider"
	"github.com/lextures/lextures/server/internal/service/badgesextraction"
	"github.com/lextures/lextures/server/internal/service/outcomesextraction"
)

// handleExtractCourseBadgesFromSyllabus is POST /api/v1/courses/{courseId}/badge-definitions/extract-from-syllabus.
// Prefers existing learning outcomes; falls back to syllabus extraction. Draft-only — does not persist.
func (d Deps) handleExtractCourseBadgesFromSyllabus() http.HandlerFunc {
	type resp struct {
		Badges []badgesextraction.DraftBadge `json:"badges"`
		Source string                        `json:"source"`
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
		if d.badgesFeatureOff(w) {
			return
		}
		courseID, ok := d.resolveCourseID(w, r)
		if !ok {
			return
		}
		viewer, ok := d.requireCourseStaffByCourseID(w, r, courseID)
		if !ok {
			return
		}

		outcomes, err := courseoutcomes.ListOutcomes(r.Context(), d.Pool, courseID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load learning outcomes.")
			return
		}

		courseCode, err := badgerepo.CourseCodeByID(r.Context(), d.Pool, courseID)
		if err != nil || strings.TrimSpace(courseCode) == "" {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to resolve course.")
			return
		}

		var syllabusMarkdown string
		if p, err := course.GetSyllabusByCourseCode(r.Context(), d.Pool, courseCode); err == nil && p != nil {
			syllabusMarkdown = outcomesextraction.SyllabusPromptMaterial(p.Sections)
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

		promptKey := syllabusMarkdown
		if len(outcomes) > 0 {
			var b strings.Builder
			for _, o := range outcomes {
				b.WriteString(o.ID.String())
				b.WriteString(o.Title)
				b.WriteString(o.Description)
			}
			promptKey = b.String() + syllabusMarkdown
		} else if syllabusMarkdown == "" {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Add learning outcomes or syllabus content before extracting badges.")
			return
		}

		if !d.enforceAIGateway(w, r, viewer, aigateway.FeatureBadgesExtraction, model, promptKey) {
			return
		}
		gwDec := aigateway.Decision{
			UserIDHash:     aigateway.UserIDHash(d.aiGatewayConfig().HMACSecret, viewer),
			OptInConfirmed: true,
		}

		bound := aiprovider.BoundCompleter{Resolver: d.aiProviderResolver(), OrgID: orgID}
		var (
			badges   []badgesextraction.DraftBadge
			callMeta aiprovider.CallMeta
			source   string
		)
		if len(outcomes) > 0 {
			inputs := make([]badgesextraction.OutcomeInput, 0, len(outcomes))
			for _, o := range outcomes {
				inputs = append(inputs, badgesextraction.OutcomeInput{
					ID:          o.ID.String(),
					Title:       o.Title,
					Description: o.Description,
				})
			}
			badges, callMeta, err = badgesextraction.ExtractFromOutcomes(r.Context(), bound, model, inputs, syllabusMarkdown)
			source = "outcomes"
		} else {
			badges, callMeta, err = badgesextraction.ExtractFromSyllabus(r.Context(), bound, model, syllabusMarkdown)
			source = "syllabus"
		}
		if err != nil {
			msg := err.Error()
			if strings.Contains(msg, "parse badges JSON") {
				writeAIGenerationFailed(w, r, "AI did not return valid badges JSON: "+msg, err)
				return
			}
			if strings.Contains(msg, "too long") {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, msg)
				return
			}
			writeAIGenerationFailed(w, r, "AI generation failed: "+msg, err)
			return
		}

		d.logAIInferenceAllowedWithProvider(r, viewer, aigateway.FeatureBadgesExtraction, model, string(callMeta.Provider), promptKey, gwDec)
		d.recordAIProviderUsage(r.Context(), AIUsageMeta{
			UserID: viewer, CourseID: &courseID, CourseCode: courseCode, Feature: aigateway.FeatureBadgesExtraction, Model: model,
		}, callMeta, true)

		if badges == nil {
			badges = []badgesextraction.DraftBadge{}
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(resp{Badges: badges, Source: source})
	}
}
