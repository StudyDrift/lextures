package httpserver

import (
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/lextures/lextures/server/internal/apierr"
	"github.com/lextures/lextures/server/internal/repos/enrollment"
	"github.com/lextures/lextures/server/internal/repos/organization"
	tutorrepo "github.com/lextures/lextures/server/internal/repos/tutor"
	tutorsessionrepo "github.com/lextures/lextures/server/internal/repos/tutorsession"
	"github.com/lextures/lextures/server/internal/repos/userai"
	aigateway "github.com/lextures/lextures/server/internal/service/aigateway"
	tutorsession "github.com/lextures/lextures/server/internal/service/tutorsession"
	lpsvc "github.com/lextures/lextures/server/internal/service/learnerprofile"
)

const aiTutorOptOutMessage = "AI tutor is disabled for your account."

func (d Deps) persistentTutorEnabled(w http.ResponseWriter) bool {
	if !d.effectiveConfig().FFPersistentTutor {
		apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Persistent AI tutor is not enabled.")
		return false
	}
	return true
}

func (d Deps) tutorSessionService() *tutorsession.Service {
	return &tutorsession.Service{Pool: d.Pool}
}

func (d Deps) enforceAITutorAccess(w http.ResponseWriter, r *http.Request, userID uuid.UUID) bool {
	if d.Pool == nil {
		apierr.WriteJSON(w, http.StatusServiceUnavailable, apierr.CodeInternal, "Database unavailable.")
		return false
	}
	optedOut, err := tutorsessionrepo.GetAITutorOptOut(r.Context(), d.Pool, userID)
	if err != nil {
		apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load AI tutor settings.")
		return false
	}
	if optedOut {
		apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, aiTutorOptOutMessage)
		return false
	}
	return true
}

func (d Deps) registerPersistentTutorRoutes(r chi.Router) {
	r.Get("/api/v1/courses/{course_code}/tutor/sessions", d.handleListTutorSessions())
	r.Post("/api/v1/courses/{course_code}/tutor/sessions", d.handleCreateTutorSession())
	r.Get("/api/v1/courses/{course_code}/tutor/sessions/{session_id}", d.handleGetTutorSession())
	r.Post("/api/v1/courses/{course_code}/tutor/sessions/{session_id}/messages", d.handlePostTutorSessionMessage())
	r.Delete("/api/v1/courses/{course_code}/tutor/sessions/{session_id}", d.handleDeleteTutorSession())
	r.Get("/api/v1/courses/{course_code}/tutor/concept-confusion", d.handleGetTutorConceptConfusion())
	r.Get("/api/v1/settings/ai-tutor-opt-out", d.handleGetAITutorOptOut())
	r.Put("/api/v1/settings/ai-tutor-opt-out", d.handlePutAITutorOptOut())
}

func (d Deps) handleListTutorSessions() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.persistentTutorEnabled(w) {
			return
		}
		courseCode := chi.URLParam(r, "course_code")
		userID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		if !d.enforceAITutorAccess(w, r, userID) {
			return
		}
		c, courseID, ok := d.tutorCourseAccess(w, r, courseCode, userID)
		if !ok {
			return
		}
		if !c.AiTutorEnabled {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "AI tutor is not enabled for this course.")
			return
		}
		sessions, err := tutorsessionrepo.ListSessions(r.Context(), d.Pool, userID, courseID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load tutor sessions.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(sessions)
	}
}

func (d Deps) handleCreateTutorSession() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.persistentTutorEnabled(w) {
			return
		}
		courseCode := chi.URLParam(r, "course_code")
		userID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		if !d.enforceAITutorAccess(w, r, userID) {
			return
		}
		c, courseID, ok := d.tutorCourseAccess(w, r, courseCode, userID)
		if !ok {
			return
		}
		if !c.AiTutorEnabled {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "AI tutor is not enabled for this course.")
			return
		}
		var req struct {
			Title *string `json:"title"`
		}
		if r.Body != nil && r.ContentLength != 0 {
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
				return
			}
		}
		session, err := tutorsessionrepo.CreateSession(r.Context(), d.Pool, userID, courseID, req.Title)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to create tutor session.")
			return
		}
		_ = d.tutorSessionService().EnsureDisclosure(r.Context(), session.ID)
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(session)
	}
}

func (d Deps) handleGetTutorSession() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.persistentTutorEnabled(w) {
			return
		}
		courseCode := chi.URLParam(r, "course_code")
		sessionID, ok := parseTutorSessionID(w, chi.URLParam(r, "session_id"))
		if !ok {
			return
		}
		userID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		if !d.enforceAITutorAccess(w, r, userID) {
			return
		}
		c, courseID, ok := d.tutorCourseAccess(w, r, courseCode, userID)
		if !ok {
			return
		}
		if !c.AiTutorEnabled {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "AI tutor is not enabled for this course.")
			return
		}
		session, err := tutorsessionrepo.GetSession(r.Context(), d.Pool, sessionID, userID, courseID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load tutor session.")
			return
		}
		if session == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Tutor session not found.")
			return
		}
		messages, err := tutorsessionrepo.ListAllMessages(r.Context(), d.Pool, sessionID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load tutor messages.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":         session.ID,
			"title":      session.Title,
			"createdAt":  session.CreatedAt,
			"lastActive": session.LastActive,
			"messages":   messages,
		})
	}
}

func (d Deps) handleDeleteTutorSession() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.persistentTutorEnabled(w) {
			return
		}
		courseCode := chi.URLParam(r, "course_code")
		sessionID, ok := parseTutorSessionID(w, chi.URLParam(r, "session_id"))
		if !ok {
			return
		}
		userID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		if !d.enforceAITutorAccess(w, r, userID) {
			return
		}
		_, courseID, ok := d.tutorCourseAccess(w, r, courseCode, userID)
		if !ok {
			return
		}
		if err := tutorsessionrepo.DeleteSession(r.Context(), d.Pool, sessionID, userID, courseID); err != nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Tutor session not found.")
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func (d Deps) handlePostTutorSessionMessage() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.persistentTutorEnabled(w) {
			return
		}
		or := d.openRouterClient()
		if or == nil || d.effectiveConfig().OpenRouterAPIKey == "" {
			apierr.WriteJSON(w, http.StatusServiceUnavailable, apierr.CodeAiNotConfigured, "AI provider not configured.")
			return
		}

		courseCode := chi.URLParam(r, "course_code")
		sessionID, ok := parseTutorSessionID(w, chi.URLParam(r, "session_id"))
		if !ok {
			return
		}
		userID, ok := d.meUserIDOrQueryToken(w, r)
		if !ok {
			return
		}
		ctx := r.Context()
		if !d.enforceAITutorAccess(w, r, userID) {
			return
		}
		c, courseID, ok := d.tutorCourseAccess(w, r, courseCode, userID)
		if !ok {
			return
		}
		if !c.AiTutorEnabled {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "AI tutor is not enabled for this course.")
			return
		}

		session, err := tutorsessionrepo.GetSession(ctx, d.Pool, sessionID, userID, courseID)
		if err != nil || session == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Tutor session not found.")
			return
		}

		var req struct {
			Content string `json:"content"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		cleaned, err := tutorsession.ValidateMessage(req.Content)
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, err.Error())
			return
		}

		orgID, err := organization.OrgIDForUser(ctx, d.Pool, userID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load user org.")
			return
		}
		budget, err := tutorrepo.GetTokenBudget(ctx, d.Pool, userID, orgID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load token budget.")
			return
		}
		if budget.TokensUsed >= budget.TokenLimit {
			apierr.WriteJSON(w, http.StatusPaymentRequired, "BUDGET_EXCEEDED",
				fmt.Sprintf("You have reached your monthly AI interaction limit of %d tokens. Your budget resets on the 1st of next month.", budget.TokenLimit))
			return
		}

		model, err := userai.GetCourseSetupModelID(ctx, d.Pool, userID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load model.")
			return
		}
		if !d.enforceAIGateway(w, r, userID, aigateway.FeatureAITutor, model, cleaned) {
			return
		}
		gwDec := aigateway.Decision{
			UserIDHash:     aigateway.UserIDHash(d.aiGatewayConfig().HMACSecret, userID),
			OptInConfirmed: true,
		}

		svc := d.tutorSessionService()
		_ = svc.EnsureDisclosure(ctx, sessionID)
		concepts, _ := svc.ListCourseConcepts(ctx, courseID)
		conceptTags := tutorsession.DetectConceptTags(cleaned, concepts)

		ragContext, citations, hasRAG := svc.RetrieveCourseRAG(ctx, courseID, c.CourseCode, c.Title, cleaned)
		history, err := tutorsessionrepo.ListRecentMessages(ctx, d.Pool, sessionID, tutorsession.HistoryMessageLimit)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load conversation history.")
			return
		}
		var profileScaffold string
		if d.profileAdaptEnabled("tutor") {
			adaptive, aerr := d.loadAdaptiveContext(ctx, userID)
			if aerr == nil && adaptive.Usable(true) && adaptive.HelpSeekingStyle != "" {
				profileScaffold = lpsvc.TutorScaffoldingPrompt(adaptive.HelpSeekingStyle)
				lpsvc.RecordAdaptation("tutor", "applied")
			} else {
				lpsvc.RecordAdaptation("tutor", "suppressed")
			}
		}
		msgs := tutorsession.BuildMessages(c.Title, history, cleaned, ragContext, hasRAG, profileScaffold)

		flusher, canFlush := w.(http.Flusher)
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.Header().Set("X-Accel-Buffering", "no")
		if canFlush {
			flusher.Flush()
		}

		start := time.Now()
		tutorsession.RecordRequest()

		if _, err := tutorsessionrepo.AppendMessage(ctx, d.Pool, sessionID, "user", cleaned, nil, conceptTags, 0); err != nil {
			tutorSSEError(w, flusher, "Failed to save your message.")
			return
		}

		fullText, streamErr := or.ChatCompletionStream(model, msgs, func(chunk string) error {
			b, _ := json.Marshal(chunk)
			_, werr := fmt.Fprintf(w, "data: {\"type\":\"content\",\"text\":%s}\n\n", string(b))
			if canFlush {
				flusher.Flush()
			}
			return werr
		})
		if streamErr != nil {
			tutorSSEError(w, flusher, "The tutor is temporarily unavailable. Your conversation history is saved.")
			return
		}

		d.logAIInferenceAllowed(r, userID, aigateway.FeatureAITutor, model, cleaned, gwDec)
		d.recordAIUsage(ctx, AIUsageMeta{
			UserID: userID, Feature: aigateway.FeatureAITutor, Model: model,
		}, fullText.Usage, true)

		validCitations := tutorsession.FilterValidCitations(nil, citations)
		if len(validCitations) == 0 && hasRAG && len(citations) > 0 {
			validCitations = citations[:1]
		}
		estimated := tutorsession.EstimateTokens(cleaned + fullText.Text)
		assistantMsg, _ := tutorsessionrepo.AppendMessage(ctx, d.Pool, sessionID, "assistant", fullText.Text, validCitations, nil, estimated)
		_ = tutorrepo.AddTokens(ctx, d.Pool, userID, orgID, estimated)
		_ = tutorsessionrepo.TouchSession(ctx, d.Pool, sessionID)

		if session.Title == nil || strings.TrimSpace(derefString(session.Title)) == "" {
			title := tutorsession.SessionTitleFromMessage(cleaned)
			_, _ = d.Pool.Exec(ctx, `UPDATE course.tutor_sessions SET title = $2 WHERE id = $1`, sessionID, title)
		}

		tutorsession.RecordLatency(tutorsession.SinceStart(start))
		tutorsession.RecordCitations(len(validCitations))

		citationsJSON, _ := json.Marshal(validCitations)
		donePayload := fmt.Sprintf(`{"type":"done","messageId":%q,"citations":%s}`, assistantMsg.ID.String(), string(citationsJSON))
		_, _ = fmt.Fprintf(w, "data: %s\n\n", donePayload)
		if canFlush {
			flusher.Flush()
		}
	}
}

func (d Deps) handleGetTutorConceptConfusion() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.persistentTutorEnabled(w) {
			return
		}
		courseCode := chi.URLParam(r, "course_code")
		userID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		c, courseID, ok := d.tutorCourseAccess(w, r, courseCode, userID)
		if !ok {
			return
		}
		if !c.AiTutorEnabled {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "AI tutor is not enabled for this course.")
			return
		}
		staffCodes, err := enrollment.ListCourseCodesWhereUserIsStaff(r.Context(), d.Pool, userID)
		if err != nil || !slices.Contains(staffCodes, courseCode) {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "Instructor access required.")
			return
		}
		since := tutorsession.ConfusionSince(time.Now().UTC())
		summary, err := tutorsessionrepo.ListConceptConfusion(r.Context(), d.Pool, courseID, since)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load concept confusion summary.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(summary)
	}
}

func (d Deps) handleGetAITutorOptOut() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.persistentTutorEnabled(w) {
			return
		}
		userID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		optedOut, err := tutorsessionrepo.GetAITutorOptOut(r.Context(), d.Pool, userID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load AI tutor settings.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{"aiTutorOptOut": optedOut})
	}
}

func (d Deps) handlePutAITutorOptOut() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.persistentTutorEnabled(w) {
			return
		}
		userID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		var body struct {
			OptOut *bool `json:"aiTutorOptOut"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.OptOut == nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "aiTutorOptOut is required.")
			return
		}
		if err := tutorsessionrepo.SetAITutorOptOut(r.Context(), d.Pool, userID, *body.OptOut); err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to save AI tutor settings.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{"aiTutorOptOut": *body.OptOut})
	}
}

func parseTutorSessionID(w http.ResponseWriter, raw string) (uuid.UUID, bool) {
	id, err := uuid.Parse(strings.TrimSpace(raw))
	if err != nil {
		apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid session id.")
		return uuid.UUID{}, false
	}
	return id, true
}

func derefString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
