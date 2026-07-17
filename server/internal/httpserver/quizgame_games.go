package httpserver

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/lextures/lextures/server/internal/apierr"
	"github.com/lextures/lextures/server/internal/courseroles"
	"github.com/lextures/lextures/server/internal/quizgame/engine"
	"github.com/lextures/lextures/server/internal/quizgame/scoring"
	"github.com/lextures/lextures/server/internal/repos/board"
	"github.com/lextures/lextures/server/internal/repos/course"
	"github.com/lextures/lextures/server/internal/repos/quizgame"
	"github.com/lextures/lextures/server/internal/telemetry"
)

const (
	quizJoinLookupLimit  = 30
	quizJoinLookupWindow = time.Minute
)

type quizJoinAttempt struct {
	count int
	start time.Time
}

var (
	quizJoinMu       sync.Mutex
	quizJoinAttempts = map[string]*quizJoinAttempt{}
)

func quizJoinRateLimited(key string) bool {
	quizJoinMu.Lock()
	defer quizJoinMu.Unlock()
	now := time.Now()
	e := quizJoinAttempts[key]
	if e == nil || now.Sub(e.start) > quizJoinLookupWindow {
		quizJoinAttempts[key] = &quizJoinAttempt{count: 1, start: now}
		return false
	}
	e.count++
	return e.count > quizJoinLookupLimit
}

func (d Deps) iqLiveHostingOff(w http.ResponseWriter) bool {
	if !d.effectiveConfig().FFIqLiveHosting {
		apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Live quiz hosting is not enabled.")
		return true
	}
	return false
}

func (d Deps) iqLiveHostingFeatureOff(w http.ResponseWriter, r *http.Request, courseCode string) bool {
	if d.iqLiveHostingOff(w) {
		return true
	}
	return d.interactiveQuizzesFeatureOff(w, r, courseCode)
}

func sessionJSON(s *quizgame.Session, includeAnswers bool) map[string]any {
	code := ""
	if s.JoinCode != nil {
		code = *s.JoinCode
	}
	questions := make([]map[string]any, 0, len(s.KitSnapshot.Questions))
	for i, q := range s.KitSnapshot.Questions {
		pub := engine.ToPublicQuestion(q, i)
		qm := map[string]any{
			"index":            pub.Index,
			"questionType":     pub.QuestionType,
			"prompt":           pub.Prompt,
			"promptMediaRef":   pub.PromptMediaRef,
			"promptMediaAlt":   pub.PromptMediaAlt,
			"options":          pub.Options,
			"timeLimitSeconds": pub.TimeLimitSeconds,
			"pointsStyle":      pub.PointsStyle,
		}
		if includeAnswers {
			qm["correctAnswer"] = q.CorrectAnswer
			correctIDs := make([]string, 0)
			for _, o := range q.Options {
				if o.IsCorrect {
					correctIDs = append(correctIDs, o.ID)
				}
			}
			qm["correctOptionIds"] = correctIDs
			qm["explanation"] = q.Explanation
		}
		questions = append(questions, qm)
	}
	out := map[string]any{
		"id":                 s.ID,
		"kitId":              s.KitID,
		"courseId":           s.CourseID,
		"hostId":             s.HostID,
		"joinCode":           code,
		"mode":               s.Mode,
		"status":             s.Status,
		"pacing":             s.Pacing,
		"phase":              s.CurrentPhase,
		"questionIndex":      s.CurrentIndex,
		"kitTitle":           s.KitSnapshot.Title,
		"questionCount":      len(s.KitSnapshot.Questions),
		"questions":          questions,
		"settings":           json.RawMessage(s.Settings),
		"scoringProfile":     s.ScoringProfile,
		"scoringProfileVer":  s.ScoringProfileVer,
		"scoringConfig":      json.RawMessage(s.ScoringConfig),
		"leaderboardPrivacy": s.LeaderboardPrivacy,
		"allowGuests":        s.AllowGuests,
		"lobbyLocked":        s.LobbyLocked,
		"namesMuted":         s.NamesMuted,
		"oneSessionRule":     s.OneSessionRule,
		"maxJoinsPerIp":      s.MaxJoinsPerIP,
		"createdAt":          s.CreatedAt.UTC().Format(time.RFC3339),
	}
	if s.QuestionOpenedAt != nil {
		out["openedAt"] = s.QuestionOpenedAt.UTC().Format(time.RFC3339Nano)
	}
	if s.QuestionDeadlineAt != nil {
		out["deadline"] = s.QuestionDeadlineAt.UTC().Format(time.RFC3339Nano)
	}
	if s.StartedAt != nil {
		out["startedAt"] = s.StartedAt.UTC().Format(time.RFC3339)
	}
	if s.EndedAt != nil {
		out["endedAt"] = s.EndedAt.UTC().Format(time.RFC3339)
	}
	return out
}

// handleCreateQuizGame is POST /api/v1/courses/{course_code}/live-quizzes/kits/{kit_id}/games.
func (d Deps) handleCreateQuizGame() http.HandlerFunc {
	type reqBody struct {
		Pacing             string              `json:"pacing"`
		Mode               string              `json:"mode"`
		Settings           json.RawMessage     `json:"settings"`
		TeamConfig         *engine.TeamConfig  `json:"teamConfig"`
		PacedConfig        *engine.PacedConfig `json:"pacedConfig"`
		ScoringProfile     string              `json:"scoringProfile"`
		ScoringConfig      scoring.Config      `json:"scoringConfig"`
		LeaderboardPrivacy string              `json:"leaderboardPrivacy"`
		PowerUpsEnabled    *bool               `json:"powerUpsEnabled"`
		AllowGuests        bool                `json:"allowGuests"`
		OneSessionRule     string              `json:"oneSessionRule"`
		MaxJoinsPerIP      int                 `json:"maxJoinsPerIp"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		courseCode, viewer, ok := d.requireCourseAccess(w, r)
		if !ok {
			return
		}
		if d.iqLiveHostingFeatureOff(w, r, courseCode) {
			return
		}
		kitID := chi.URLParam(r, "kit_id")
		hasPerm, err := courseroles.UserHasPermission(r.Context(), d.Pool, viewer, "course:"+courseCode+":item:create")
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to verify permissions.")
			return
		}
		if !hasPerm {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "You do not have permission to host a live quiz.")
			return
		}
		var body reqBody
		if r.Body != nil {
			_ = json.NewDecoder(r.Body).Decode(&body)
		}
		mode := engine.NormalizeMode(body.Mode)
		if mode == engine.ModeHomework {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Use the assignments API to start homework mode.")
			return
		}
		if !d.iqModeAllowed(mode) {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "This game mode is not enabled.")
			return
		}
		if err := quizgame.CheckConcurrentGamesQuota(r.Context(), d.Pool, courseCode); err != nil {
			if errors.Is(err, quizgame.ErrConcurrentGamesQuota) {
				telemetry.RecordBusinessEvent("quizgame.quota.concurrent_games_rejected")
				apierr.WriteJSON(w, http.StatusConflict, apierr.CodeConflict, "Concurrent live-game quota reached. End an active game or ask an admin to raise the limit.")
				return
			}
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not verify game quotas.")
			return
		}
		allowGuests := body.AllowGuests
		if allowGuests {
			if !d.effectiveConfig().FFIqGuestJoin {
				allowGuests = false
			} else if d.effectiveConfig().CoppaWorkflowEnabled {
				if hasMinors, merr := board.CourseHasEnrolledMinors(r.Context(), d.Pool, courseCode); merr == nil && hasMinors {
					allowGuests = false
				}
			}
			if allowGuests {
				if eff, eerr := quizgame.ResolveEffectiveSettingsForCourse(r.Context(), d.Pool, courseCode); eerr == nil {
					if eff.GuestJoinPolicy == quizgame.GuestJoinDisabled {
						allowGuests = false
					}
				}
			}
		}
		sess, err := quizgame.CreateGame(r.Context(), d.Pool, quizgame.CreateGameInput{
			CourseCode:         courseCode,
			KitID:              kitID,
			HostID:             viewer,
			Pacing:             body.Pacing,
			Mode:               string(mode),
			Settings:           body.Settings,
			TeamConfig:         body.TeamConfig,
			PacedConfig:        body.PacedConfig,
			ScoringProfile:     body.ScoringProfile,
			ScoringConfig:      body.ScoringConfig,
			LeaderboardPrivacy: body.LeaderboardPrivacy,
			PowerUpsEnabled:    body.PowerUpsEnabled,
			AllowGuests:        allowGuests,
			OneSessionRule:     body.OneSessionRule,
			MaxJoinsPerIP:      body.MaxJoinsPerIP,
		})
		if errors.Is(err, quizgame.ErrKitNotReady) {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Kit is not ready to host. Fix validation issues first.")
			return
		}
		if errors.Is(err, quizgame.ErrSessionNotFound) {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Kit not found.")
			return
		}
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not start game.")
			return
		}
		if mode == engine.ModeTeam {
			_, _ = quizgame.CreateTeams(r.Context(), d.Pool, sess.ID, nil, nil)
		}
		telemetry.RecordBusinessEvent("quizgame.game.create")
		code := ""
		if sess.JoinCode != nil {
			code = *sess.JoinCode
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"gameId":   sess.ID,
			"joinCode": code,
			"game":     sessionJSON(sess, true),
		})
	}
}

// handleGetQuizGame is GET /api/v1/courses/{course_code}/live-quizzes/games/{game_id}.
func (d Deps) handleGetQuizGame() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		courseCode, viewer, ok := d.requireCourseAccess(w, r)
		if !ok {
			return
		}
		if d.iqLiveHostingFeatureOff(w, r, courseCode) {
			return
		}
		gameID := chi.URLParam(r, "game_id")
		sess, err := quizgame.GetSessionByCourse(r.Context(), d.Pool, courseCode, gameID)
		if errors.Is(err, quizgame.ErrSessionNotFound) || sess == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Game not found.")
			return
		}
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not load game.")
			return
		}
		isHost := sess.HostID != nil && *sess.HostID == viewer.String()
		hasPerm, _ := courseroles.UserHasPermission(r.Context(), d.Pool, viewer, "course:"+courseCode+":item:create")
		canHost := isHost || hasPerm
		players, _ := quizgame.ListPlayers(r.Context(), d.Pool, gameID)
		playerOut := make([]map[string]any, 0, len(players))
		for _, p := range players {
			m := map[string]any{
				"id":         p.ID,
				"nickname":   p.Nickname,
				"totalScore": p.TotalScore,
				"streak":     p.Streak,
				"connected":  p.Connected,
				"teamId":     p.TeamID,
			}
			if p.FinishedAt != nil {
				m["finished"] = true
			}
			if engine.NormalizeMode(sess.Mode) == engine.ModeStudentPaced || engine.NormalizeMode(sess.Mode) == engine.ModeHomework {
				m["currentIndex"] = p.CurrentIndex
				m["currentPhase"] = p.CurrentPhase
			}
			playerOut = append(playerOut, m)
		}
		out := sessionJSON(sess, canHost)
		out["players"] = playerOut
		if engine.NormalizeMode(sess.Mode) == engine.ModeTeam {
			if board, err := quizgame.RefreshTeamScores(r.Context(), d.Pool, gameID); err == nil {
				out["teamLeaderboard"] = board
			}
			if teams, err := quizgame.ListTeams(r.Context(), d.Pool, gameID); err == nil {
				teamOut := make([]map[string]any, 0, len(teams))
				for _, t := range teams {
					teamOut = append(teamOut, teamJSON(t))
				}
				out["teams"] = teamOut
			}
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(out)
	}
}

// handleGetQuizGameLeaderboard is GET .../games/{game_id}/leaderboard (IQ.5).
func (d Deps) handleGetQuizGameLeaderboard() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		courseCode, _, ok := d.requireCourseAccess(w, r)
		if !ok {
			return
		}
		if d.iqLiveHostingFeatureOff(w, r, courseCode) {
			return
		}
		gameID := chi.URLParam(r, "game_id")
		sess, err := quizgame.GetSessionByCourse(r.Context(), d.Pool, courseCode, gameID)
		if errors.Is(err, quizgame.ErrSessionNotFound) || sess == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Game not found.")
			return
		}
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not load game.")
			return
		}
		view, err := quizgame.BuildLeaderboardView(r.Context(), d.Pool, sess, 50, "")
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not load leaderboard.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(view)
	}
}

// handleGetQuizGamePlayerResponses is GET .../games/{game_id}/responses/{player_id} (IQ.5 / IQ.7).
func (d Deps) handleGetQuizGamePlayerResponses() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		courseCode, viewer, ok := d.requireCourseAccess(w, r)
		if !ok {
			return
		}
		if d.iqLiveHostingFeatureOff(w, r, courseCode) {
			return
		}
		gameID := chi.URLParam(r, "game_id")
		playerID := chi.URLParam(r, "player_id")
		sess, err := quizgame.GetSessionByCourse(r.Context(), d.Pool, courseCode, gameID)
		if errors.Is(err, quizgame.ErrSessionNotFound) || sess == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Game not found.")
			return
		}
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not load game.")
			return
		}
		isHost := sess.HostID != nil && *sess.HostID == viewer.String()
		hasPerm, _ := courseroles.UserHasPermission(r.Context(), d.Pool, viewer, "course:"+courseCode+":item:create")
		player, perr := quizgame.GetPlayer(r.Context(), d.Pool, playerID)
		if perr != nil || player == nil || player.SessionID != gameID {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Player not found.")
			return
		}
		isSelf := player.UserID != nil && *player.UserID == viewer.String()
		if !isHost && !hasPerm && !isSelf {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "You do not have permission to view these responses.")
			return
		}
		rows, err := quizgame.ListPlayerResponses(r.Context(), d.Pool, gameID, playerID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not load responses.")
			return
		}
		out := make([]map[string]any, 0, len(rows))
		for _, resp := range rows {
			item := map[string]any{
				"questionIndex": resp.QuestionIndex,
				"isCorrect":     resp.IsCorrect,
				"responseMs":    resp.ResponseMs,
				"points":        resp.Points,
				"answeredAt":    resp.AnsweredAt.UTC().Format(time.RFC3339Nano),
			}
			if len(resp.PointsBreakdown) > 0 {
				item["pointsBreakdown"] = json.RawMessage(resp.PointsBreakdown)
			}
			if isHost || hasPerm {
				item["answer"] = json.RawMessage(resp.Answer)
			}
			out = append(out, item)
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"playerId":  playerID,
			"nickname":  player.Nickname,
			"totalScore": player.TotalScore,
			"responses": out,
		})
	}
}

// handleEndQuizGame is POST /api/v1/courses/{course_code}/live-quizzes/games/{game_id}/end.
func (d Deps) handleEndQuizGame() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		courseCode, viewer, ok := d.requireCourseAccess(w, r)
		if !ok {
			return
		}
		if d.iqLiveHostingFeatureOff(w, r, courseCode) {
			return
		}
		hasPerm, err := courseroles.UserHasPermission(r.Context(), d.Pool, viewer, "course:"+courseCode+":item:create")
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to verify permissions.")
			return
		}
		if !hasPerm {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "You do not have permission to end this game.")
			return
		}
		gameID := chi.URLParam(r, "game_id")
		sess, err := quizgame.GetSessionByCourse(r.Context(), d.Pool, courseCode, gameID)
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
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not end game.")
			return
		}
		if _, rerr := quizgame.BuildAndStoreReport(r.Context(), d.Pool, ended.ID); rerr != nil {
			telemetry.RecordBusinessEvent("quizgame.report.build_failed")
		} else {
			telemetry.RecordBusinessEvent("quizgame.report.generated")
		}
		broadcastQuizGameState(r.Context(), d, ended)
		telemetry.RecordBusinessEvent("quizgame.game.end")
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(sessionJSON(ended, true))
	}
}

// handleJoinQuizLookup is GET /api/v1/live-quizzes/join/{code} (public, rate-limited).
func (d Deps) handleJoinQuizLookup() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		if d.iqLiveHostingOff(w) {
			return
		}
		if quizJoinRateLimited("lookup:" + r.RemoteAddr) {
			apierr.WriteJSON(w, http.StatusTooManyRequests, apierr.CodeRateLimited, "Too many join lookups. Please try again later.")
			return
		}
		code := strings.TrimSpace(chi.URLParam(r, "code"))
		if code == "" || len(code) > 12 {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Game not found.")
			return
		}
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
		courseCode, err := course.GetCourseCodeByID(r.Context(), d.Pool, courseID)
		if err != nil || courseCode == nil || *courseCode == "" {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not look up game.")
			return
		}
		allowsGuests := d.guestJoinAllowed(r, *courseCode, sess)
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"gameId":       sess.ID,
			"courseCode":   *courseCode,
			"kitTitle":     sess.KitSnapshot.Title,
			"requiresAuth": !allowsGuests,
			"allowsGuests": allowsGuests,
			"lobbyLocked":  sess.LobbyLocked,
			"phase":        sess.CurrentPhase,
			"status":       sess.Status,
		})
	}
}

// handleJoinQuizPlayer is POST /api/v1/courses/{course_code}/live-quizzes/games/{game_id}/players.
// IQ.4/IQ.9: enrolled join + rejoin with nickname moderation, one-session, and join limits.
func (d Deps) handleJoinQuizPlayer() http.HandlerFunc {
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
		if quizJoinRateLimited("join:" + r.RemoteAddr) {
			apierr.WriteJSON(w, http.StatusTooManyRequests, apierr.CodeRateLimited, "Too many join attempts. Please try again later.")
			return
		}
		courseCode, viewer, ok := d.requireCourseAccess(w, r)
		if !ok {
			return
		}
		if d.iqLiveHostingFeatureOff(w, r, courseCode) {
			return
		}
		gameID := chi.URLParam(r, "game_id")
		sess, err := quizgame.GetSessionByCourse(r.Context(), d.Pool, courseCode, gameID)
		if errors.Is(err, quizgame.ErrSessionNotFound) || sess == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Game not found.")
			return
		}
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not load game.")
			return
		}
		if sess.Status == "ended" || sess.Status == "abandoned" {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "This game has ended.")
			return
		}
		var body reqBody
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Nickname is required.")
			return
		}
		// Rejoins do not consume a new player slot; check quota only for new joins.
		existing, _ := quizgame.GetPlayerByUser(r.Context(), d.Pool, gameID, viewer)
		if existing == nil {
			if err := quizgame.CheckPlayersPerGameQuota(r.Context(), d.Pool, courseCode, gameID); err != nil {
				if errors.Is(err, quizgame.ErrPlayersPerGameQuota) {
					telemetry.RecordBusinessEvent("quizgame.quota.players_rejected")
					apierr.WriteJSON(w, http.StatusConflict, apierr.CodeConflict, "This game is full (player quota reached).")
					return
				}
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not verify player quotas.")
				return
			}
		}
		res, err := quizgame.AddPlayer(r.Context(), d.Pool, quizgame.AddPlayerInput{
			SessionID:  gameID,
			UserID:     &viewer,
			Nickname:   body.Nickname,
			ClientMeta: body.ClientMeta,
			RemoteIP:   clientRemoteIP(r),
		})
		if errors.Is(err, quizgame.ErrNicknameDenied) {
			_ = quizgame.RecordSafetyEvent(r.Context(), d.Pool, sess.ID, nil, &viewer, quizgame.SafetyNicknameDenied, map[string]any{
				"nickname": body.Nickname,
			})
		}
		if writeQuizJoinError(w, err) {
			return
		}
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Could not join game.")
			return
		}
		if res.Rejoined {
			telemetry.RecordBusinessEvent("quizgame.player.rejoin")
		} else {
			telemetry.RecordBusinessEvent("quizgame.player.join")
		}
		broadcastQuizGameState(r.Context(), d, sess)
		status := http.StatusCreated
		if res.Rejoined {
			status = http.StatusOK
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"playerId":    res.Player.ID,
			"nickname":    res.Player.Nickname,
			"playerToken": res.PlayerToken,
			"totalScore":  res.Player.TotalScore,
			"streak":      res.Player.Streak,
			"rejoined":    res.Rejoined,
		})
	}
}
