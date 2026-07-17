package httpserver

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/lextures/lextures/server/internal/apierr"
	"github.com/lextures/lextures/server/internal/repos/organization"
	"github.com/lextures/lextures/server/internal/repos/quizgame"
	svcadmindaudit "github.com/lextures/lextures/server/internal/service/adminaudit"
	"github.com/lextures/lextures/server/internal/service/notifications"
	"github.com/lextures/lextures/server/internal/telemetry"
)

func (d Deps) auditIQAdmin(r *http.Request, actor uuid.UUID, eventType string, targetType string, targetID *uuid.UUID, after any) {
	orgID, _ := organization.OrgIDForUser(r.Context(), d.Pool, actor)
	var orgPtr *uuid.UUID
	if orgID != uuid.Nil {
		orgPtr = &orgID
	}
	var afterBytes []byte
	if after != nil {
		afterBytes, _ = json.Marshal(after)
	}
	tt := targetType
	_, _ = svcadmindaudit.Record(r.Context(), d.Pool, svcadmindaudit.RecordParams{
		OrgID:      orgPtr,
		EventType:  eventType,
		ActorID:    actor,
		TargetType: &tt,
		TargetID:   targetID,
		AfterValue: afterBytes,
	})
}

func (d Deps) resolveAdminOrgID(w http.ResponseWriter, r *http.Request, actor uuid.UUID) (uuid.UUID, bool) {
	orgID, err := organization.OrgIDForUser(r.Context(), d.Pool, actor)
	if err != nil {
		apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to resolve organization.")
		return uuid.Nil, false
	}
	if q := r.URL.Query().Get("orgId"); q != "" {
		parsed, perr := uuid.Parse(q)
		if perr != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid orgId.")
			return uuid.Nil, false
		}
		orgID = parsed
	}
	if orgID == uuid.Nil {
		apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Organization required.")
		return uuid.Nil, false
	}
	return orgID, true
}

// handleAdminIQSettings is GET/PATCH /api/v1/admin/settings/interactive-quizzes (IQ.11).
func (d Deps) handleAdminIQSettings() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		actor, ok := d.adminRbacUser(w, r)
		if !ok {
			return
		}
		switch r.Method {
		case http.MethodGet:
			settings, err := quizgame.GetPlatformSettings(r.Context(), d.Pool)
			if err != nil {
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load Live Quiz settings.")
				return
			}
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			_ = json.NewEncoder(w).Encode(settings)
		case http.MethodPatch:
			var body struct {
				MaxConcurrentGames        *int    `json:"maxConcurrentGames"`
				ClearMaxConcurrentGames   bool    `json:"clearMaxConcurrentGames"`
				MaxPlayersPerGame         *int    `json:"maxPlayersPerGame"`
				MaxKitsPerCourse          *int    `json:"maxKitsPerCourse"`
				ClearMaxKitsPerCourse     bool    `json:"clearMaxKitsPerCourse"`
				RetentionDays             *int    `json:"retentionDays"`
				GuestJoinPolicy           *string `json:"guestJoinPolicy"`
				DefaultMode               *string `json:"defaultMode"`
				DefaultLeaderboardPrivacy *string `json:"defaultLeaderboardPrivacy"`
				AIGenerationEnabled       *bool   `json:"aiGenerationEnabled"`
				AIGenerationsPerDay       *int    `json:"aiGenerationsPerDay"`
				ClearAIGenerationsPerDay  bool    `json:"clearAiGenerationsPerDay"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
				return
			}
			settings, err := quizgame.PatchPlatformSettings(r.Context(), d.Pool, quizgame.PatchPlatformSettingsInput{
				MaxConcurrentGames:        body.MaxConcurrentGames,
				ClearMaxConcurrentGames:   body.ClearMaxConcurrentGames,
				MaxPlayersPerGame:         body.MaxPlayersPerGame,
				MaxKitsPerCourse:          body.MaxKitsPerCourse,
				ClearMaxKitsPerCourse:     body.ClearMaxKitsPerCourse,
				RetentionDays:             body.RetentionDays,
				GuestJoinPolicy:           body.GuestJoinPolicy,
				DefaultMode:               body.DefaultMode,
				DefaultLeaderboardPrivacy: body.DefaultLeaderboardPrivacy,
				AIGenerationEnabled:       body.AIGenerationEnabled,
				AIGenerationsPerDay:       body.AIGenerationsPerDay,
				ClearAIGenerationsPerDay:  body.ClearAIGenerationsPerDay,
			})
			if err != nil {
				if strings.Contains(err.Error(), "quizgame:") {
					apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, err.Error())
					return
				}
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to update Live Quiz settings.")
				return
			}
			d.auditIQAdmin(r, actor, "quizgame.admin.settings_updated", "quizgame_settings", nil, settings)
			telemetry.RecordBusinessEvent("quizgame.admin.settings_updated")
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			_ = json.NewEncoder(w).Encode(settings)
		default:
			w.Header().Set("Allow", "GET, PATCH")
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		}
	}
}

// handleAdminOrgIQSettings is GET/PATCH /api/v1/admin/org-units/{id}/interactive-quizzes (IQ.11).
func (d Deps) handleAdminOrgIQSettings() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		actor, ok := d.adminRbacUser(w, r)
		if !ok {
			return
		}
		orgID, err := uuid.Parse(chi.URLParam(r, "id"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid org id.")
			return
		}
		switch r.Method {
		case http.MethodGet:
			eff, err := quizgame.ResolveEffectiveSettings(r.Context(), d.Pool, orgID)
			if err != nil {
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load org Live Quiz settings.")
				return
			}
			stored, _ := quizgame.GetOrgSettings(r.Context(), d.Pool, orgID)
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"effective": eff,
				"overrides": stored,
			})
		case http.MethodPatch:
			var body quizgame.OrgOverrides
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
				return
			}
			eff, err := quizgame.UpsertOrgSettings(r.Context(), d.Pool, orgID, &actor, body)
			if err != nil {
				if strings.Contains(err.Error(), "quizgame:") {
					apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, err.Error())
					return
				}
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to update org Live Quiz settings.")
				return
			}
			d.auditIQAdmin(r, actor, "quizgame.admin.org_settings_updated", "quizgame_org_settings", &orgID, eff)
			telemetry.RecordBusinessEvent("quizgame.admin.org_settings_updated")
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			_ = json.NewEncoder(w).Encode(eff)
		default:
			w.Header().Set("Allow", "GET, PATCH")
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		}
	}
}

// handleAdminIQAnalytics is GET /api/v1/admin/interactive-quizzes/analytics (IQ.11).
func (d Deps) handleAdminIQAnalytics() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		actor, ok := d.adminRbacUser(w, r)
		if !ok {
			return
		}
		orgID, ok := d.resolveAdminOrgID(w, r, actor)
		if !ok {
			return
		}
		to := time.Now().UTC()
		from := to.AddDate(0, 0, -30)
		if raw := r.URL.Query().Get("from"); raw != "" {
			if t, err := time.Parse("2006-01-02", raw); err == nil {
				from = t.UTC()
			}
		}
		if raw := r.URL.Query().Get("to"); raw != "" {
			if t, err := time.Parse("2006-01-02", raw); err == nil {
				to = t.UTC().Add(24 * time.Hour)
			}
		}
		sum, err := quizgame.GetAnalytics(r.Context(), d.Pool, orgID, from, to)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load Live Quiz analytics.")
			return
		}
		telemetry.RecordBusinessEvent("quizgame.admin.analytics_viewed")
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(sum)
	}
}

// handleAdminIQReviewQueue is GET /api/v1/admin/interactive-quizzes/review-queue (IQ.11).
func (d Deps) handleAdminIQReviewQueue() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, ok := d.adminRbacUser(w, r); !ok {
			return
		}
		status := r.URL.Query().Get("status")
		limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
		items, err := quizgame.ListReviewQueue(r.Context(), d.Pool, status, limit)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load review queue.")
			return
		}
		pending, _ := quizgame.CountPendingReviews(r.Context(), d.Pool)
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"items":          items,
			"pendingCount":   pending,
		})
	}
}

// handleAdminIQReviewAction is POST .../review-queue/{id}/{approve|reject|action} (IQ.11).
func (d Deps) handleAdminIQReviewAction() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		actor, ok := d.adminRbacUser(w, r)
		if !ok {
			return
		}
		id := chi.URLParam(r, "id")
		action := strings.ToLower(chi.URLParam(r, "action"))
		var body struct {
			Reason string `json:"reason"`
		}
		if r.Body != nil {
			_ = json.NewDecoder(r.Body).Decode(&body)
		}
		var item *quizgame.ModerationQueueItem
		var err error
		switch action {
		case "approve":
			item, err = quizgame.ApproveReview(r.Context(), d.Pool, id, actor)
		case "reject":
			item, err = quizgame.RejectReview(r.Context(), d.Pool, id, actor, body.Reason)
		case "action", "takedown":
			item, err = quizgame.ActionReview(r.Context(), d.Pool, id, actor, body.Reason)
		default:
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Unknown review action.")
			return
		}
		if item == nil && err == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Review item not found.")
			return
		}
		if err != nil {
			if strings.Contains(err.Error(), "quizgame:") {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, err.Error())
				return
			}
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to process review.")
			return
		}
		tid, _ := uuid.Parse(item.ID)
		d.auditIQAdmin(r, actor, "quizgame.admin.review_"+action, "quizgame_review", &tid, item)
		telemetry.RecordBusinessEvent("quizgame.admin.review_" + action)

		if action == "reject" && item.Submitter != nil {
			if sid, perr := uuid.Parse(*item.Submitter); perr == nil {
				title := "Live Quiz catalog submission rejected"
				msg := "Your kit was not listed in the public catalog."
				if item.Reason != nil && *item.Reason != "" {
					msg = "Your kit was not listed: " + *item.Reason
				}
				_ = d.pushNotificationService().Enqueue(r.Context(), sid, notifications.EventInboxMessage, title, msg, "/admin/live-quizzes")
			}
		}

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(item)
	}
}

// handleAdminIQLiveGames is GET /api/v1/admin/interactive-quizzes/games (IQ.11).
func (d Deps) handleAdminIQLiveGames() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		actor, ok := d.adminRbacUser(w, r)
		if !ok {
			return
		}
		orgID, ok := d.resolveAdminOrgID(w, r, actor)
		if !ok {
			return
		}
		games, err := quizgame.ListLiveGames(r.Context(), d.Pool, orgID, 50)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to list live games.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{"games": games})
	}
}

// handleAdminIQForceEnd is POST /api/v1/admin/interactive-quizzes/games/{game_id}/force-end (IQ.11).
func (d Deps) handleAdminIQForceEnd() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		actor, ok := d.adminRbacUser(w, r)
		if !ok {
			return
		}
		gameID := chi.URLParam(r, "game_id")
		sess, err := quizgame.GetSession(r.Context(), d.Pool, gameID)
		if errors.Is(err, quizgame.ErrSessionNotFound) || sess == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Game not found.")
			return
		}
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not load game.")
			return
		}
		ended, err := quizgame.EndSession(r.Context(), d.Pool, gameID, time.Now().UTC())
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not force-end game.")
			return
		}
		if _, rerr := quizgame.BuildAndStoreReport(r.Context(), d.Pool, ended.ID); rerr != nil {
			telemetry.RecordBusinessEvent("quizgame.report.build_failed")
		}
		broadcastQuizGameState(r.Context(), d, ended)
		tid, _ := uuid.Parse(ended.ID)
		d.auditIQAdmin(r, actor, "quizgame.admin.force_end", "quizgame_session", &tid, map[string]any{
			"gameId": ended.ID,
			"status": ended.Status,
		})
		telemetry.RecordBusinessEvent("quizgame.admin.force_end")
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(sessionJSON(ended, true))
	}
}

// handleAdminIQBulkArchiveKits is POST /api/v1/admin/interactive-quizzes/kits/bulk-archive (IQ.11).
func (d Deps) handleAdminIQBulkArchiveKits() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		actor, ok := d.adminRbacUser(w, r)
		if !ok {
			return
		}
		orgID, ok := d.resolveAdminOrgID(w, r, actor)
		if !ok {
			return
		}
		var body struct {
			OlderThanDays int `json:"olderThanDays"`
			Limit         int `json:"limit"`
		}
		if r.Body != nil {
			_ = json.NewDecoder(r.Body).Decode(&body)
		}
		if body.OlderThanDays <= 0 {
			body.OlderThanDays = 365
		}
		cutoff := time.Now().UTC().AddDate(0, 0, -body.OlderThanDays)
		n, err := quizgame.BulkArchiveKits(r.Context(), d.Pool, orgID, cutoff, body.Limit)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to bulk-archive kits.")
			return
		}
		d.auditIQAdmin(r, actor, "quizgame.admin.bulk_archive_kits", "quizgame_kits", &orgID, map[string]any{
			"archived":      n,
			"olderThanDays": body.OlderThanDays,
		})
		telemetry.RecordBusinessEvent("quizgame.admin.bulk_archive_kits")
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{"archived": n})
	}
}
