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
	cfg := d.effectiveConfig()
	if !cfg.FFInteractiveQuizzes || !cfg.FFIqLiveHosting {
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
		"id":           s.ID,
		"kitId":        s.KitID,
		"courseId":     s.CourseID,
		"hostId":       s.HostID,
		"joinCode":     code,
		"mode":         s.Mode,
		"status":       s.Status,
		"pacing":       s.Pacing,
		"phase":        s.CurrentPhase,
		"questionIndex": s.CurrentIndex,
		"kitTitle":     s.KitSnapshot.Title,
		"questionCount": len(s.KitSnapshot.Questions),
		"questions":    questions,
		"settings":     json.RawMessage(s.Settings),
		"createdAt":    s.CreatedAt.UTC().Format(time.RFC3339),
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
		Pacing   string          `json:"pacing"`
		Settings json.RawMessage `json:"settings"`
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
		sess, err := quizgame.CreateGame(r.Context(), d.Pool, quizgame.CreateGameInput{
			CourseCode: courseCode,
			KitID:      kitID,
			HostID:     viewer,
			Pacing:     body.Pacing,
			Settings:   body.Settings,
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
			playerOut = append(playerOut, map[string]any{
				"id":         p.ID,
				"nickname":   p.Nickname,
				"totalScore": p.TotalScore,
				"streak":     p.Streak,
				"connected":  p.Connected,
			})
		}
		out := sessionJSON(sess, canHost)
		out["players"] = playerOut
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(out)
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
		// Guest join ships with IQ.9; enrolled-only until then.
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"gameId":       sess.ID,
			"courseCode":   *courseCode,
			"kitTitle":     sess.KitSnapshot.Title,
			"requiresAuth": true,
			"allowsGuests": false,
			"phase":        sess.CurrentPhase,
			"status":       sess.Status,
		})
	}
}

// handleJoinQuizPlayer is POST /api/v1/courses/{course_code}/live-quizzes/games/{game_id}/players.
// IQ.4: enrolled join + rejoin (same user rotates reconnect token). Guest join deferred to IQ.9.
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
		res, err := quizgame.AddPlayer(r.Context(), d.Pool, quizgame.AddPlayerInput{
			SessionID:  gameID,
			UserID:     &viewer,
			Nickname:   body.Nickname,
			ClientMeta: body.ClientMeta,
		})
		if errors.Is(err, quizgame.ErrPlayerExists) {
			apierr.WriteJSON(w, http.StatusConflict, apierr.CodeConflict, "Nickname is already taken in this game.")
			return
		}
		if errors.Is(err, quizgame.ErrNicknameInvalid) {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Nickname is invalid. Use 1–24 letters, numbers, or spaces.")
			return
		}
		if errors.Is(err, quizgame.ErrSessionEnded) {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "This game has ended.")
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
