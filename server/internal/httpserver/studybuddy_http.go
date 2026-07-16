package httpserver

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/lextures/lextures/server/internal/apierr"
	"github.com/lextures/lextures/server/internal/repos/course"
	"github.com/lextures/lextures/server/internal/repos/learnergoals"
	userrepo "github.com/lextures/lextures/server/internal/repos/user"
	"github.com/lextures/lextures/server/internal/repos/userai"
	studybuddyrepo "github.com/lextures/lextures/server/internal/repos/studybuddy"
	aigateway "github.com/lextures/lextures/server/internal/service/aigateway"
	"github.com/lextures/lextures/server/internal/service/studybuddy"
)

func (d Deps) aiStudyBuddyEnabled(w http.ResponseWriter) bool {
	if !d.effectiveConfig().FFAIStudyBuddy {
		apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "AI study buddy is not enabled.")
		return false
	}
	return true
}

func (d Deps) studyBuddyService() *studybuddy.Service {
	return &studybuddy.Service{Pool: d.Pool, Config: d.effectiveConfig()}
}

func (d Deps) registerStudyBuddyRoutes(r chi.Router) {
	r.Get("/api/v1/courses/{course_code}/study-buddy/memory", d.handleGetStudyBuddyMemory())
	r.Delete("/api/v1/courses/{course_code}/study-buddy/memory", d.handleDeleteStudyBuddyMemory())
	r.Get("/api/v1/courses/{course_code}/study-buddy/prompts", d.handleGetStudyBuddyPrompts())
	r.Post("/api/v1/courses/{course_code}/study-buddy/message", d.handlePostStudyBuddyMessage())
}

func (d Deps) studyBuddyCourseAccess(w http.ResponseWriter, r *http.Request, courseCode string, userID uuid.UUID) (*course.CoursePublic, uuid.UUID, bool) {
	return d.tutorCourseAccess(w, r, courseCode, userID)
}

func (d Deps) handleGetStudyBuddyMemory() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.aiStudyBuddyEnabled(w) {
			return
		}
		courseCode := chi.URLParam(r, "course_code")
		userID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		c, courseID, ok := d.studyBuddyCourseAccess(w, r, courseCode, userID)
		if !ok {
			return
		}
		_ = c
		summary, err := d.studyBuddyService().GetMemorySummary(r.Context(), userID, courseID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load study buddy memory.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(summary)
	}
}

func (d Deps) handleDeleteStudyBuddyMemory() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.aiStudyBuddyEnabled(w) {
			return
		}
		courseCode := chi.URLParam(r, "course_code")
		userID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		_, courseID, ok := d.studyBuddyCourseAccess(w, r, courseCode, userID)
		if !ok {
			return
		}
		if err := d.studyBuddyService().ClearMemory(r.Context(), userID, courseID); err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to clear study buddy memory.")
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func (d Deps) handleGetStudyBuddyPrompts() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.aiStudyBuddyEnabled(w) {
			return
		}
		courseCode := chi.URLParam(r, "course_code")
		userID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		_, courseID, ok := d.studyBuddyCourseAccess(w, r, courseCode, userID)
		if !ok {
			return
		}
		prompts, err := d.studyBuddyService().ListPrompts(r.Context(), userID, courseID, time.Now().UTC())
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load study buddy prompts.")
			return
		}
		if prompts == nil {
			prompts = []studybuddy.Prompt{}
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{"prompts": prompts})
	}
}

func (d Deps) handlePostStudyBuddyMessage() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.aiStudyBuddyEnabled(w) {
			return
		}
		// AI provider is checked before auth to preserve historical 503 (vs 401)
		// behavior for misconfigured deployments; org-scoped resolution happens below.
		if !d.aiConfigured(r.Context(), nil) {
			apierr.WriteJSON(w, http.StatusServiceUnavailable, apierr.CodeAiNotConfigured, aiNotConfiguredMsg)
			return
		}
		courseCode := chi.URLParam(r, "course_code")
		userID, ok := d.meUserIDOrQueryToken(w, r)
		if !ok {
			return
		}
		ctx := r.Context()
		orgID := d.orgIDPtrForUser(ctx, userID)
		c, courseID, ok := d.studyBuddyCourseAccess(w, r, courseCode, userID)
		if !ok {
			return
		}

		var req struct {
			Message   string `json:"message"`
			SessionID string `json:"sessionId"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		cleaned, err := studybuddy.ValidateMessage(req.Message)
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, err.Error())
			return
		}

		sessionID := uuid.Nil
		if strings.TrimSpace(req.SessionID) != "" {
			sessionID, err = uuid.Parse(strings.TrimSpace(req.SessionID))
			if err != nil {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid sessionId.")
				return
			}
		}

		model, err := d.resolveStudyBuddyModel(ctx, userID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load model.")
			return
		}
		if !d.enforceAIGateway(w, r, userID, aigateway.FeatureAIStudyBuddy, model, cleaned) {
			return
		}
		gwDec := aigateway.Decision{
			UserIDHash:     aigateway.UserIDHash(d.aiGatewayConfig().HMACSecret, userID),
			OptInConfirmed: true,
		}

		memory, err := d.studyBuddyService().RefreshMemory(ctx, userID, courseID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load study buddy context.")
			return
		}
		session, err := studybuddyrepo.GetOrCreateSession(ctx, d.Pool, userID, courseID, sessionID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load chat session.")
			return
		}

		displayName, priorLevel := d.studyBuddyLearnerProfile(ctx, userID)
		ragContext, citations, hasRAG := d.studyBuddyService().RetrieveCourseRAG(ctx, courseID, c.CourseCode, c.Title, cleaned)
		msgs := d.studyBuddyService().BuildMessages(c.Title, memory, priorLevel, displayName, session.Messages, cleaned, ragContext, hasRAG)

		flusher, canFlush := w.(http.Flusher)
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.Header().Set("X-Accel-Buffering", "no")
		if canFlush {
			flusher.Flush()
		}

		if err := studybuddyrepo.AppendSessionMessage(ctx, d.Pool, session.ID, studybuddyrepo.Message{Role: "user", Content: cleaned}); err != nil {
			studyBuddySSEError(w, flusher, "Failed to save your message.")
			return
		}

		fullText, callMeta, streamErr := d.completeStreamOrBuffered(ctx, orgID, model, msgs, func(chunk string) error {
			b, _ := json.Marshal(chunk)
			_, werr := fmt.Fprintf(w, "data: {\"type\":\"content\",\"text\":%s}\n\n", string(b))
			if canFlush {
				flusher.Flush()
			}
			return werr
		})
		if streamErr != nil {
			studyBuddySSEError(w, flusher, "Study buddy is temporarily unavailable.")
			return
		}

		d.logAIInferenceAllowedWithProvider(r, userID, aigateway.FeatureAIStudyBuddy, model, string(callMeta.Provider), cleaned, gwDec)
		d.recordAIProviderResult(ctx, AIUsageMeta{
			UserID: userID, Feature: aigateway.FeatureAIStudyBuddy, Model: model,
		}, callMeta, fullText, true)

		_ = studybuddyrepo.AppendSessionMessage(ctx, d.Pool, session.ID, studybuddyrepo.Message{Role: "assistant", Content: fullText.Text})

		history, _ := studybuddyrepo.ListSessionMessages(ctx, d.Pool, session.ID)
		summary := studybuddy.SummarizeSession(history)
		if summary != "" {
			_ = studybuddyrepo.UpdateSessionSummary(ctx, d.Pool, userID, courseID, summary)
		}

		citationsJSON, _ := json.Marshal(citations)
		donePayload := fmt.Sprintf(`{"type":"done","sessionId":%q,"citations":%s}`, session.ID.String(), string(citationsJSON))
		_, _ = fmt.Fprintf(w, "data: %s\n\n", donePayload)
		if canFlush {
			flusher.Flush()
		}
	}
}

func (d Deps) resolveStudyBuddyModel(ctx context.Context, userID uuid.UUID) (string, error) {
	return userai.GetCourseSetupModelID(ctx, d.Pool, userID)
}

func (d Deps) studyBuddyLearnerProfile(ctx context.Context, userID uuid.UUID) (displayName, priorLevel string) {
	priorLevel = "beginner"
	if u, err := userrepo.FindByID(ctx, d.Pool, userID); err == nil && u != nil && u.DisplayName != nil {
		displayName = strings.TrimSpace(*u.DisplayName)
	}
	if goals, err := learnergoals.Get(ctx, d.Pool, userID); err == nil && goals != nil {
		if strings.TrimSpace(goals.PriorKnowledgeLevel) != "" {
			priorLevel = goals.PriorKnowledgeLevel
		}
	}
	return displayName, priorLevel
}

func studyBuddySSEError(w http.ResponseWriter, flusher http.Flusher, msg string) {
	b, _ := json.Marshal(msg)
	_, _ = fmt.Fprintf(w, "data: {\"type\":\"error\",\"message\":%s}\n\n", string(b))
	if flusher != nil {
		flusher.Flush()
	}
}
