package httpserver

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/lextures/lextures/server/internal/apierr"
	"github.com/lextures/lextures/server/internal/repos/learnergoals"
	"github.com/lextures/lextures/server/internal/repos/organization"
	"github.com/lextures/lextures/server/internal/service/onboarding"
)

func (d Deps) onboardingFlowEnabled(w http.ResponseWriter) bool {
	if !d.effectiveConfig().FFOnboardingFlow {
		apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Onboarding is not enabled.")
		return false
	}
	return true
}

func (d Deps) registerOnboardingGoalsRoutes(r chi.Router) {
	r.Get("/api/v1/me/onboarding-status", d.handleGetOnboardingStatus())
	r.Post("/api/v1/me/onboarding", d.handlePostOnboarding())
	r.Get("/api/v1/me/onboarding/diagnostic-questions", d.handleGetOnboardingDiagnosticQuestions())
	r.Get("/api/v1/me/goals", d.handleGetMyGoals())
	r.Patch("/api/v1/me/goals", d.handlePatchMyGoals())
}

type onboardingStatusResponse struct {
	Completed      bool `json:"completed"`
	Step           int  `json:"step"`
	ShouldShowFlow bool `json:"shouldShowFlow"`
}

func (d Deps) handleGetOnboardingStatus() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		if !d.onboardingFlowEnabled(w) {
			return
		}
		userID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		row, err := learnergoals.Get(r.Context(), d.Pool, userID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not load onboarding status.")
			return
		}
		resp := onboardingStatusResponse{ShouldShowFlow: true}
		if row != nil {
			resp.Completed = row.OnboardingCompleted
			resp.Step = row.OnboardingStep
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(resp)
	}
}

type postOnboardingBody struct {
	Step                *int           `json:"step"`
	Topic               *string        `json:"topic"`
	GoalText            *string        `json:"goalText"`
	TargetDate          *string        `json:"targetDate"`
	DailyMinutes        *int           `json:"dailyMinutes"`
	PriorKnowledgeLevel *string        `json:"priorKnowledgeLevel"`
	DiagnosticAnswers   map[string]int `json:"diagnosticAnswers"`
	SkipDiagnostic      *bool          `json:"skipDiagnostic"`
	ReminderOptIn       *bool          `json:"reminderOptIn"`
	ReminderTime        *string        `json:"reminderTime"`
	Complete            *bool          `json:"complete"`
	SkipAll             *bool          `json:"skipAll"`
	TermsAccepted       *bool          `json:"termsAccepted"`
}

func (d Deps) handlePostOnboarding() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		if !d.onboardingFlowEnabled(w) {
			return
		}
		userID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		payload, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Could not read body.")
			return
		}
		var body postOnboardingBody
		if err := json.Unmarshal(payload, &body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}

		ctx := r.Context()
		patch := learnergoals.StepPatch{}

		if body.SkipAll != nil && *body.SkipAll {
			done := true
			step := 6
			patch.Step = &step
			patch.OnboardingCompleted = &done
			row, err := learnergoals.ApplyStep(ctx, d.Pool, userID, patch)
			if err != nil {
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not save onboarding.")
				return
			}
			d.writeGoalsJSON(w, row)
			return
		}

		if body.Step != nil {
			patch.Step = body.Step
		}
		if body.Topic != nil {
			t := strings.TrimSpace(*body.Topic)
			patch.Topic = &t
		}
		if body.GoalText != nil {
			g := strings.TrimSpace(*body.GoalText)
			patch.GoalText = &g
		}
		if body.TargetDate != nil {
			raw := strings.TrimSpace(*body.TargetDate)
			if raw == "" {
				patch.ClearTargetDate = true
			} else {
				parsed, err := time.Parse("2006-01-02", raw)
				if err != nil {
					apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "targetDate must be YYYY-MM-DD.")
					return
				}
				patch.TargetDate = &parsed
			}
		}
		if body.DailyMinutes != nil {
			if *body.DailyMinutes < 5 || *body.DailyMinutes > 480 {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "dailyMinutes must be between 5 and 480.")
				return
			}
			patch.DailyMinutes = body.DailyMinutes
		}
		if body.PriorKnowledgeLevel != nil {
			patch.PriorKnowledgeLevel = body.PriorKnowledgeLevel
		}
		if body.SkipDiagnostic != nil && *body.SkipDiagnostic {
			skipped := true
			patch.DiagnosticSkipped = &skipped
		}
		if len(body.DiagnosticAnswers) > 0 {
			existing, _ := learnergoals.Get(ctx, d.Pool, userID)
			topic := "general"
			if existing != nil && existing.Topic != "" {
				topic = existing.Topic
			}
			if body.Topic != nil && strings.TrimSpace(*body.Topic) != "" {
				topic = strings.TrimSpace(*body.Topic)
			}
			score := onboarding.ScoreDiagnostic(topic, body.DiagnosticAnswers)
			patch.DiagnosticScore = &score
			skipped := false
			patch.DiagnosticSkipped = &skipped
		}
		if body.ReminderOptIn != nil {
			patch.ReminderOptIn = body.ReminderOptIn
		}
		if body.ReminderTime != nil {
			rt := strings.TrimSpace(*body.ReminderTime)
			patch.ReminderTime = &rt
		}

		row, err := learnergoals.ApplyStep(ctx, d.Pool, userID, patch)
		if err != nil {
			if strings.Contains(err.Error(), "prior knowledge") {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, err.Error())
				return
			}
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not save onboarding.")
			return
		}

		if body.Complete != nil && *body.Complete {
			if body.TermsAccepted != nil && !*body.TermsAccepted {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Terms of service must be accepted.")
				return
			}
			row, err = d.completeOnboarding(ctx, userID, row)
			if err != nil {
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not complete onboarding.")
				return
			}
		}

		d.writeGoalsJSON(w, row)
	}
}

func (d Deps) completeOnboarding(ctx context.Context, userID uuid.UUID, row *learnergoals.Row) (*learnergoals.Row, error) {
	level := onboarding.EffectiveLevel(row.PriorKnowledgeLevel, row.DiagnosticScore, row.DiagnosticSkipped)
	var rec *learnergoals.RecommendedCourse
	orgID, err := organization.OrgIDForUser(ctx, d.Pool, userID)
	if err == nil {
		code, title, ok := onboarding.RecommendCourse(ctx, d.Pool, orgID, row.Topic, level)
		if ok {
			rec = &learnergoals.RecommendedCourse{Code: code, Title: title}
		}
	}

	done := true
	step := 6
	patch := learnergoals.StepPatch{
		Step:                &step,
		OnboardingCompleted: &done,
		RecommendedCourse:   rec,
	}
	return learnergoals.ApplyStep(ctx, d.Pool, userID, patch)
}

func (d Deps) handleGetOnboardingDiagnosticQuestions() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		if !d.onboardingFlowEnabled(w) {
			return
		}
		if _, ok := d.meUserID(w, r); !ok {
			return
		}
		topic := strings.TrimSpace(r.URL.Query().Get("topic"))
		if topic == "" {
			topic = "general"
		}
		questions := onboarding.QuestionsForTopic(topic)
		type qOut struct {
			ID      string   `json:"id"`
			Prompt  string   `json:"prompt"`
			Choices []string `json:"choices"`
		}
		out := make([]qOut, 0, len(questions))
		for _, q := range questions {
			out = append(out, qOut{ID: q.ID, Prompt: q.Prompt, Choices: q.Choices})
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{"questions": out})
	}
}

func (d Deps) handleGetMyGoals() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		userID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		row, err := learnergoals.Get(r.Context(), d.Pool, userID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not load goals.")
			return
		}
		if row == nil {
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			_ = json.NewEncoder(w).Encode(map[string]any{"goals": nil})
			return
		}
		d.writeGoalsJSON(w, row)
	}
}

func (d Deps) handlePatchMyGoals() http.HandlerFunc {
	type body struct {
		Topic               *string `json:"topic"`
		GoalText            *string `json:"goalText"`
		TargetDate          *string `json:"targetDate"`
		DailyMinutes        *int    `json:"dailyMinutes"`
		PriorKnowledgeLevel *string `json:"priorKnowledgeLevel"`
		ReminderOptIn       *bool   `json:"reminderOptIn"`
		ReminderTime        *string `json:"reminderTime"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			w.Header().Set("Allow", http.MethodPatch)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		userID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		payload, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Could not read body.")
			return
		}
		var b body
		if err := json.Unmarshal(payload, &b); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		patch := learnergoals.GoalsPatch{}
		if b.Topic != nil {
			t := strings.TrimSpace(*b.Topic)
			patch.Topic = &t
		}
		if b.GoalText != nil {
			g := strings.TrimSpace(*b.GoalText)
			patch.GoalText = &g
		}
		if b.TargetDate != nil {
			raw := strings.TrimSpace(*b.TargetDate)
			if raw == "" {
				patch.ClearTargetDate = true
			} else {
				parsed, err := time.Parse("2006-01-02", raw)
				if err != nil {
					apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "targetDate must be YYYY-MM-DD.")
					return
				}
				patch.TargetDate = &parsed
			}
		}
		if b.DailyMinutes != nil {
			if *b.DailyMinutes < 5 || *b.DailyMinutes > 480 {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "dailyMinutes must be between 5 and 480.")
				return
			}
			patch.DailyMinutes = b.DailyMinutes
		}
		if b.PriorKnowledgeLevel != nil {
			patch.PriorKnowledgeLevel = b.PriorKnowledgeLevel
		}
		if b.ReminderOptIn != nil {
			patch.ReminderOptIn = b.ReminderOptIn
		}
		if b.ReminderTime != nil {
			rt := strings.TrimSpace(*b.ReminderTime)
			patch.ReminderTime = &rt
		}
		row, err := learnergoals.PatchGoals(r.Context(), d.Pool, userID, patch)
		if err != nil {
			if strings.Contains(err.Error(), "prior knowledge") {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, err.Error())
				return
			}
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not save goals.")
			return
		}
		d.writeGoalsJSON(w, row)
	}
}

func (d Deps) writeGoalsJSON(w http.ResponseWriter, row *learnergoals.Row) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(map[string]any{"goals": row})
}
