package httpserver

import (
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/lextures/lextures/server/internal/apierr"
	"github.com/lextures/lextures/server/internal/courseroles"
	"github.com/lextures/lextures/server/internal/repos/board"
	"github.com/lextures/lextures/server/internal/repos/course"
	"github.com/lextures/lextures/server/internal/repos/quizgame"
	"github.com/lextures/lextures/server/internal/telemetry"
)

func clientRemoteIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		return strings.TrimSpace(parts[0])
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// guestJoinAllowed reports whether guest join is permitted for this course/session.
// Under-13 (COPPA) courses always block public guest join — teacher-mediated enrolled only.
func (d Deps) guestJoinAllowed(r *http.Request, courseCode string, sess *quizgame.Session) bool {
	cfg := d.effectiveConfig()
	if !cfg.FFInteractiveQuizzes || !cfg.FFIqLiveHosting || !cfg.FFIqGuestJoin {
		return false
	}
	if sess == nil || !sess.AllowGuests {
		return false
	}
	if cfg.CoppaWorkflowEnabled {
		hasMinors, err := board.CourseHasEnrolledMinors(r.Context(), d.Pool, courseCode)
		if err == nil && hasMinors {
			return false
		}
	}
	if eff, err := quizgame.ResolveEffectiveSettingsForCourse(r.Context(), d.Pool, courseCode); err == nil {
		if eff.GuestJoinPolicy == quizgame.GuestJoinDisabled {
			return false
		}
	}
	return true
}

func (d Deps) requireQuizGameHost(w http.ResponseWriter, r *http.Request) (courseCode string, viewer uuid.UUID, sess *quizgame.Session, ok bool) {
	courseCode, viewer, ok = d.requireCourseAccess(w, r)
	if !ok {
		return "", uuid.Nil, nil, false
	}
	if d.iqLiveHostingFeatureOff(w, r, courseCode) {
		return "", uuid.Nil, nil, false
	}
	gameID := chi.URLParam(r, "game_id")
	sess, err := quizgame.GetSessionByCourse(r.Context(), d.Pool, courseCode, gameID)
	if errors.Is(err, quizgame.ErrSessionNotFound) || sess == nil {
		apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Game not found.")
		return "", uuid.Nil, nil, false
	}
	if err != nil {
		apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not load game.")
		return "", uuid.Nil, nil, false
	}
	hasPerm, err := courseroles.UserHasPermission(r.Context(), d.Pool, viewer, "course:"+courseCode+":item:create")
	if err != nil {
		apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to verify permissions.")
		return "", uuid.Nil, nil, false
	}
	if !hasPerm {
		// Host of the session may also control safety.
		if sess.HostID == nil || *sess.HostID != viewer.String() {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "You do not have permission to moderate this game.")
			return "", uuid.Nil, nil, false
		}
	}
	return courseCode, viewer, sess, true
}

func writeQuizJoinError(w http.ResponseWriter, err error) bool {
	switch {
	case errors.Is(err, quizgame.ErrPlayerExists):
		apierr.WriteJSON(w, http.StatusConflict, apierr.CodeConflict, "Nickname is already taken in this game.")
	case errors.Is(err, quizgame.ErrNicknameInvalid):
		apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Nickname is invalid. Use 1–24 letters, numbers, or spaces.")
	case errors.Is(err, quizgame.ErrNicknameDenied):
		apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "That nickname isn’t allowed. Please choose another.")
	case errors.Is(err, quizgame.ErrLobbyLocked):
		apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "The lobby is locked. New players cannot join.")
	case errors.Is(err, quizgame.ErrPlayerBanned):
		apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "You cannot rejoin this game.")
	case errors.Is(err, quizgame.ErrJoinLimitExceeded):
		apierr.WriteJSON(w, http.StatusTooManyRequests, apierr.CodeRateLimited, "Too many joins from this network. Please try again later.")
	case errors.Is(err, quizgame.ErrOneSessionRefuse):
		apierr.WriteJSON(w, http.StatusConflict, apierr.CodeConflict, "You are already connected in another session.")
	case errors.Is(err, quizgame.ErrGuestsNotAllowed):
		apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "Guest join is not enabled for this game.")
	case errors.Is(err, quizgame.ErrSessionEnded):
		apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "This game has ended.")
	default:
		return false
	}
	return true
}

// handleKickQuizPlayer is POST .../players/{player_id}/kick
func (d Deps) handleKickQuizPlayer() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		_, viewer, sess, ok := d.requireQuizGameHost(w, r)
		if !ok {
			return
		}
		playerID := chi.URLParam(r, "player_id")
		if err := quizgame.KickPlayer(r.Context(), d.Pool, sess.ID, playerID); err != nil {
			if errors.Is(err, quizgame.ErrPlayerNotFound) {
				apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Player not found.")
				return
			}
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not kick player.")
			return
		}
		pid, _ := uuid.Parse(playerID)
		_ = quizgame.RecordSafetyEvent(r.Context(), d.Pool, sess.ID, &pid, &viewer, quizgame.SafetyKicked, map[string]any{"banned": true})
		gameUUID, _ := uuid.Parse(sess.ID)
		notifyQuizPlayerKicked(r.Context(), gameUUID, playerID)
		broadcastQuizGameState(r.Context(), d, sess)
		telemetry.RecordBusinessEvent("quizgame.player.kick")
		w.WriteHeader(http.StatusNoContent)
	}
}

// handleBanQuizPlayer is POST .../players/{player_id}/ban (alias of kick+ban).
func (d Deps) handleBanQuizPlayer() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		_, viewer, sess, ok := d.requireQuizGameHost(w, r)
		if !ok {
			return
		}
		playerID := chi.URLParam(r, "player_id")
		if err := quizgame.BanPlayer(r.Context(), d.Pool, sess.ID, playerID); err != nil {
			if errors.Is(err, quizgame.ErrPlayerNotFound) {
				apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Player not found.")
				return
			}
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not ban player.")
			return
		}
		pid, _ := uuid.Parse(playerID)
		_ = quizgame.RecordSafetyEvent(r.Context(), d.Pool, sess.ID, &pid, &viewer, quizgame.SafetyBanned, map[string]any{})
		gameUUID, _ := uuid.Parse(sess.ID)
		notifyQuizPlayerKicked(r.Context(), gameUUID, playerID)
		broadcastQuizGameState(r.Context(), d, sess)
		telemetry.RecordBusinessEvent("quizgame.player.ban")
		w.WriteHeader(http.StatusNoContent)
	}
}

// handleRenameQuizPlayer is POST .../players/{player_id}/rename
func (d Deps) handleRenameQuizPlayer() http.HandlerFunc {
	type reqBody struct {
		Nickname string `json:"nickname"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		_, viewer, sess, ok := d.requireQuizGameHost(w, r)
		if !ok {
			return
		}
		playerID := chi.URLParam(r, "player_id")
		var body reqBody
		_ = json.NewDecoder(r.Body).Decode(&body)
		nick := strings.TrimSpace(body.Nickname)
		if nick == "" {
			var err error
			nick, err = quizgame.NeutralPlayerName(r.Context(), d.Pool, sess.ID, playerID)
			if err != nil {
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not rename player.")
				return
			}
		}
		if err := quizgame.RenamePlayer(r.Context(), d.Pool, sess.ID, playerID, nick); err != nil {
			if writeQuizJoinError(w, err) {
				return
			}
			if errors.Is(err, quizgame.ErrPlayerNotFound) {
				apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Player not found.")
				return
			}
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Could not rename player.")
			return
		}
		pid, _ := uuid.Parse(playerID)
		_ = quizgame.RecordSafetyEvent(r.Context(), d.Pool, sess.ID, &pid, &viewer, quizgame.SafetyRenamed, map[string]any{"nickname": nick})
		live, _ := quizgame.GetSession(r.Context(), d.Pool, sess.ID)
		if live != nil {
			broadcastQuizGameState(r.Context(), d, live)
		}
		telemetry.RecordBusinessEvent("quizgame.player.rename")
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{"playerId": playerID, "nickname": nick})
	}
}

// handlePatchQuizGameSafety is PATCH .../games/{game_id}/safety
func (d Deps) handlePatchQuizGameSafety() http.HandlerFunc {
	type reqBody struct {
		AllowGuests    *bool   `json:"allowGuests"`
		OneSessionRule *string `json:"oneSessionRule"`
		MaxJoinsPerIP  *int    `json:"maxJoinsPerIp"`
		LobbyLocked    *bool   `json:"lobbyLocked"`
		NamesMuted     *bool   `json:"namesMuted"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			w.Header().Set("Allow", http.MethodPatch)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		_, viewer, sess, ok := d.requireQuizGameHost(w, r)
		if !ok {
			return
		}
		var body reqBody
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid body.")
			return
		}
		if body.AllowGuests != nil && *body.AllowGuests {
			// Guests also need platform flag; under-13 courses cannot enable.
			if !d.effectiveConfig().FFIqGuestJoin {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Guest join is not enabled on this platform.")
				return
			}
			courseCode := chi.URLParam(r, "course_code")
			if d.effectiveConfig().CoppaWorkflowEnabled {
				hasMinors, err := board.CourseHasEnrolledMinors(r.Context(), d.Pool, courseCode)
				if err == nil && hasMinors {
					apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden,
						"Guest join is not allowed for courses with minors. Use enrolled-only play.")
					return
				}
			}
		}
		if err := quizgame.PatchSessionSafety(r.Context(), d.Pool, sess.ID, quizgame.SessionSafetyPatch{
			AllowGuests:    body.AllowGuests,
			OneSessionRule: body.OneSessionRule,
			MaxJoinsPerIP:  body.MaxJoinsPerIP,
			LobbyLocked:    body.LobbyLocked,
			NamesMuted:     body.NamesMuted,
		}); err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not update safety settings.")
			return
		}
		if body.NamesMuted != nil {
			_ = quizgame.RecordSafetyEvent(r.Context(), d.Pool, sess.ID, nil, &viewer, quizgame.SafetyMuted, map[string]any{"muted": *body.NamesMuted})
		}
		if body.LobbyLocked != nil {
			_ = quizgame.RecordSafetyEvent(r.Context(), d.Pool, sess.ID, nil, &viewer, quizgame.SafetyLobbyLocked, map[string]any{"locked": *body.LobbyLocked})
		}
		live, _ := quizgame.GetSession(r.Context(), d.Pool, sess.ID)
		if live == nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not reload game.")
			return
		}
		broadcastQuizGameState(r.Context(), d, live)
		telemetry.RecordBusinessEvent("quizgame.safety.patch")
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"allowGuests":    live.AllowGuests,
			"lobbyLocked":    live.LobbyLocked,
			"namesMuted":     live.NamesMuted,
			"oneSessionRule": live.OneSessionRule,
			"maxJoinsPerIp":  live.MaxJoinsPerIP,
		})
	}
}

// handleFlagQuizGameContent is POST .../games/{game_id}/flag
func (d Deps) handleFlagQuizGameContent() http.HandlerFunc {
	type reqBody struct {
		PlayerID      *string `json:"playerId"`
		QuestionIndex *int    `json:"questionIndex"`
		Reason        string  `json:"reason"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		_, viewer, sess, ok := d.requireQuizGameHost(w, r)
		if !ok {
			return
		}
		var body reqBody
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil || strings.TrimSpace(body.Reason) == "" {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Reason is required.")
			return
		}
		var pid *uuid.UUID
		if body.PlayerID != nil && *body.PlayerID != "" {
			parsed, err := uuid.Parse(*body.PlayerID)
			if err != nil {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid playerId.")
				return
			}
			pid = &parsed
		}
		detail := map[string]any{"reason": strings.TrimSpace(body.Reason)}
		if body.QuestionIndex != nil {
			detail["questionIndex"] = *body.QuestionIndex
		}
		if err := quizgame.RecordSafetyEvent(r.Context(), d.Pool, sess.ID, pid, &viewer, quizgame.SafetyContentFlag, detail); err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not record flag.")
			return
		}
		telemetry.RecordBusinessEvent("quizgame.safety.flag")
		w.WriteHeader(http.StatusNoContent)
	}
}

// handleListQuizGameSafetyEvents is GET .../games/{game_id}/safety-events
func (d Deps) handleListQuizGameSafetyEvents() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		_, _, sess, ok := d.requireQuizGameHost(w, r)
		if !ok {
			return
		}
		events, err := quizgame.ListSafetyEvents(r.Context(), d.Pool, sess.ID, 200)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not load safety events.")
			return
		}
		flags, _ := quizgame.ComputeIntegrityFlags(r.Context(), d.Pool, sess.ID)
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"events":         events,
			"integrityFlags": flags,
		})
	}
}

// handleJoinQuizGuest is POST /api/v1/live-quizzes/join/{code}/players (public guest join).
func (d Deps) handleJoinQuizGuest() http.HandlerFunc {
	type reqBody struct {
		Nickname   string          `json:"nickname"`
		ClientMeta json.RawMessage `json:"clientMeta"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		if d.iqLiveHostingOff(w) {
			return
		}
		if quizJoinRateLimited("guestjoin:" + r.RemoteAddr) {
			apierr.WriteJSON(w, http.StatusTooManyRequests, apierr.CodeRateLimited, "Too many join attempts. Please try again later.")
			return
		}
		code := strings.TrimSpace(chi.URLParam(r, "code"))
		sess, err := quizgame.LookupByJoinCode(r.Context(), d.Pool, code)
		if errors.Is(err, quizgame.ErrJoinCodeNotFound) || sess == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Game not found.")
			return
		}
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not look up game.")
			return
		}
		courseID, err := uuid.Parse(sess.CourseID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not look up game.")
			return
		}
		courseCodePtr, err := course.GetCourseCodeByID(r.Context(), d.Pool, courseID)
		if err != nil || courseCodePtr == nil || *courseCodePtr == "" {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not look up game.")
			return
		}
		courseCode := *courseCodePtr
		if !d.guestJoinAllowed(r, courseCode, sess) {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "Guest join is not available for this game.")
			return
		}
		var body reqBody
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Nickname is required.")
			return
		}
		if err := quizgame.CheckPlayersPerGameQuota(r.Context(), d.Pool, courseCode, sess.ID); err != nil {
			if errors.Is(err, quizgame.ErrPlayersPerGameQuota) {
				telemetry.RecordBusinessEvent("quizgame.quota.players_rejected")
				apierr.WriteJSON(w, http.StatusConflict, apierr.CodeConflict, "This game is full (player quota reached).")
				return
			}
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not verify player quotas.")
			return
		}
		res, err := quizgame.AddPlayer(r.Context(), d.Pool, quizgame.AddPlayerInput{
			SessionID:  sess.ID,
			UserID:     nil,
			Nickname:   body.Nickname,
			ClientMeta: body.ClientMeta,
			RemoteIP:   clientRemoteIP(r),
			AllowGuest: true,
		})
		if errors.Is(err, quizgame.ErrNicknameDenied) {
			_ = quizgame.RecordSafetyEvent(r.Context(), d.Pool, sess.ID, nil, nil, quizgame.SafetyNicknameDenied, map[string]any{
				"nickname": body.Nickname, "guest": true,
			})
		}
		if writeQuizJoinError(w, err) {
			return
		}
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Could not join game.")
			return
		}
		telemetry.RecordBusinessEvent("quizgame.player.guest_join")
		broadcastQuizGameState(r.Context(), d, sess)
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"playerId":    res.Player.ID,
			"nickname":    res.Player.Nickname,
			"playerToken": res.PlayerToken,
			"totalScore":  res.Player.TotalScore,
			"streak":      res.Player.Streak,
			"rejoined":    false,
			"isGuest":     true,
			"courseCode":  courseCode,
			"gameId":      sess.ID,
		})
	}
}
