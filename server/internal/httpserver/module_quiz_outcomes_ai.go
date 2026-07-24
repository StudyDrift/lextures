package httpserver

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/lextures/lextures/server/internal/apierr"
	"github.com/lextures/lextures/server/internal/repos/coursemodulequizzes"
	"github.com/lextures/lextures/server/internal/repos/courseoutcomes"
	userai "github.com/lextures/lextures/server/internal/repos/user"
	aigateway "github.com/lextures/lextures/server/internal/service/aigateway"
	"github.com/lextures/lextures/server/internal/service/aiprovider"
	"github.com/lextures/lextures/server/internal/service/quizoutcomesmapping"
)

// handleSuggestQuizOutcomeLinks is POST .../quizzes/{item_id}/suggest-outcome-links.
// Returns draft mappings only; does not persist.
func (d Deps) handleSuggestQuizOutcomeLinks() http.HandlerFunc {
	type questionBody struct {
		ID     string `json:"id"`
		Prompt string `json:"prompt"`
	}
	type reqBody struct {
		Questions []questionBody `json:"questions"`
	}
	type resp struct {
		Suggestions []quizoutcomesmapping.DraftSuggestion `json:"suggestions"`
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
		courseCode, courseID, itemID, viewer, ok := d.requireQuizItemEdit(w, r)
		if !ok {
			return
		}

		var body reqBody
		if r.Body != nil {
			_ = json.NewDecoder(r.Body).Decode(&body)
		}

		quiz, err := coursemodulequizzes.GetForCourseItem(r.Context(), d.Pool, courseID, itemID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load quiz.")
			return
		}
		if quiz == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Not found.")
			return
		}

		outcomes, err := courseoutcomes.ListOutcomes(r.Context(), d.Pool, courseID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load outcomes.")
			return
		}
		if len(outcomes) == 0 {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Add learning outcomes before suggesting mappings.")
			return
		}

		outcomeInputs := make([]quizoutcomesmapping.OutcomeInput, 0, len(outcomes))
		for _, o := range outcomes {
			outcomeInputs = append(outcomeInputs, quizoutcomesmapping.OutcomeInput{
				ID:          o.ID.String(),
				Title:       o.Title,
				Description: o.Description,
			})
		}

		questionInputs := make([]quizoutcomesmapping.QuestionInput, 0)
		if len(body.Questions) > 0 {
			for _, q := range body.Questions {
				id := strings.TrimSpace(q.ID)
				if id == "" {
					continue
				}
				questionInputs = append(questionInputs, quizoutcomesmapping.QuestionInput{
					ID:     id,
					Prompt: q.Prompt,
				})
			}
		} else if !quiz.IsAdaptive {
			for _, q := range quiz.Questions {
				id := strings.TrimSpace(q.ID)
				if id == "" {
					continue
				}
				questionInputs = append(questionInputs, quizoutcomesmapping.QuestionInput{
					ID:     id,
					Prompt: q.Prompt,
				})
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

		suggestIn := quizoutcomesmapping.SuggestInput{
			QuizTitle: quiz.Title,
			QuizIntro: quiz.Markdown,
			Outcomes:  outcomeInputs,
			Questions: questionInputs,
		}
		// Approximate prompt size for gateway metering.
		meterMaterial := quiz.Title + "\n" + quiz.Markdown
		for _, o := range outcomeInputs {
			meterMaterial += "\n" + o.Title
		}
		for _, q := range questionInputs {
			meterMaterial += "\n" + q.Prompt
		}

		if !d.enforceAIGateway(w, r, viewer, aigateway.FeatureQuizOutcomeMapping, model, meterMaterial) {
			return
		}
		gwDec := aigateway.Decision{
			UserIDHash:     aigateway.UserIDHash(d.aiGatewayConfig().HMACSecret, viewer),
			OptInConfirmed: true,
		}

		bound := aiprovider.BoundCompleter{Resolver: d.aiProviderResolver(), OrgID: orgID}
		suggestions, callMeta, err := quizoutcomesmapping.Suggest(
			r.Context(), bound, model, quizoutcomesmapping.DefaultSystemPrompt, suggestIn,
		)
		if err != nil {
			msg := err.Error()
			if strings.Contains(msg, "parse outcome mapping JSON") {
				writeAIGenerationFailed(w, r, "AI did not return valid outcome mapping JSON: "+msg, err)
				return
			}
			if strings.Contains(msg, "too long") {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, msg)
				return
			}
			writeAIGenerationFailed(w, r, "AI generation failed: "+msg, err)
			return
		}

		// Drop suggestions that already exist for this quiz item.
		existing, err := courseoutcomes.ListLinksForCourse(r.Context(), d.Pool, courseID)
		if err == nil {
			type key struct {
				kind, qid, oid, m, i string
			}
			have := make(map[key]struct{})
			for _, link := range existing {
				if link.StructureItemID != itemID {
					continue
				}
				have[key{
					kind: link.TargetKind,
					qid:  link.QuizQuestionID,
					oid:  link.OutcomeID.String(),
					m:    link.MeasurementLevel,
					i:    link.IntensityLevel,
				}] = struct{}{}
			}
			filtered := suggestions[:0]
			for _, s := range suggestions {
				k := key{kind: s.TargetKind, qid: s.QuizQuestionID, oid: s.OutcomeID, m: s.MeasurementLevel, i: s.IntensityLevel}
				if _, ok := have[k]; ok {
					continue
				}
				filtered = append(filtered, s)
			}
			suggestions = filtered
		}

		d.logAIInferenceAllowedWithProvider(r, viewer, aigateway.FeatureQuizOutcomeMapping, model, string(callMeta.Provider), meterMaterial, gwDec)
		d.recordAIProviderUsage(r.Context(), AIUsageMeta{
			UserID: viewer, CourseID: &courseID, CourseCode: courseCode, Feature: aigateway.FeatureQuizOutcomeMapping, Model: model,
		}, callMeta, true)

		if suggestions == nil {
			suggestions = []quizoutcomesmapping.DraftSuggestion{}
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(resp{Suggestions: suggestions})
	}
}
