package httpserver

import (
	"context"
	"encoding/json"
	"errors"
	"math/rand"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/coder/websocket"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/lextures/lextures/server/internal/apierr"
	"github.com/lextures/lextures/server/internal/courseroles"
	"github.com/lextures/lextures/server/internal/quizgame/engine"
	"github.com/lextures/lextures/server/internal/quizgame/scoring"
	"github.com/lextures/lextures/server/internal/repos/course"
	"github.com/lextures/lextures/server/internal/repos/enrollment"
	"github.com/lextures/lextures/server/internal/repos/quizgame"
	"github.com/lextures/lextures/server/internal/telemetry"
	"github.com/lextures/lextures/server/internal/yrelay"
)

const (
	quizGameWSMaxMessageBytes = 64 << 10 // 64 KiB
	quizGameWSMaxMsgsPerSec   = 20
	quizGameWSBurst           = 40
)

var globalQuizGameRooms = yrelay.NewRegistry()

type quizGamePeerMeta struct {
	Role     string // host | projector | player
	PlayerID string
	GameID   string
}

var (
	quizGamePeerMu   sync.RWMutex
	quizGamePeerByID = map[uuid.UUID]quizGamePeerMeta{}
)

func setQuizGamePeer(id uuid.UUID, meta quizGamePeerMeta) {
	quizGamePeerMu.Lock()
	quizGamePeerByID[id] = meta
	quizGamePeerMu.Unlock()
}

func clearQuizGamePeer(id uuid.UUID) {
	quizGamePeerMu.Lock()
	delete(quizGamePeerByID, id)
	quizGamePeerMu.Unlock()
}

func getQuizGamePeer(id uuid.UUID) (quizGamePeerMeta, bool) {
	quizGamePeerMu.RLock()
	defer quizGamePeerMu.RUnlock()
	m, ok := quizGamePeerByID[id]
	return m, ok
}

// handleQuizGameWS is GET /api/v1/courses/{course_code}/live-quizzes/games/{game_id}/ws.
// First message: {"authToken":"...","role":"host"|"projector"|"player","playerToken":"..."}.
func (d Deps) handleQuizGameWS() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		if d.JWTSigner == nil || d.Pool == nil {
			apierr.WriteJSON(w, http.StatusServiceUnavailable, apierr.CodeInternal, "server misconfiguration")
			return
		}
		if !d.effectiveConfig().FFIqLiveHosting {
			http.Error(w, "live quiz hosting disabled", http.StatusNotFound)
			return
		}

		courseCode := chi.URLParam(r, "course_code")
		gameIDRaw := chi.URLParam(r, "game_id")
		gameID, err := uuid.Parse(gameIDRaw)
		if err != nil {
			http.Error(w, "invalid game id", http.StatusBadRequest)
			return
		}

		conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{OriginPatterns: []string{"*"}})
		if err != nil {
			return
		}
		defer func() { _ = conn.Close(websocket.StatusNormalClosure, "") }()

		authCtx, cancelAuth := context.WithTimeout(r.Context(), 2*time.Minute)
		defer cancelAuth()
		typ, b, err := conn.Read(authCtx)
		if err != nil || typ != websocket.MessageText {
			return
		}
		var m struct {
			AuthToken   string `json:"authToken"`
			Role        string `json:"role"`
			PlayerToken string `json:"playerToken"`
		}
		if err := json.Unmarshal(b, &m); err != nil {
			return
		}
		role := strings.ToLower(strings.TrimSpace(m.Role))
		if role == "" {
			role = "host"
		}

		crow, err := course.GetPublicByCourseCode(r.Context(), d.Pool, courseCode)
		if err != nil || crow == nil || !crow.InteractiveQuizzesEnabled {
			return
		}
		sess, err := quizgame.GetSessionByCourse(r.Context(), d.Pool, courseCode, gameID.String())
		if err != nil || sess == nil {
			return
		}

		var userID uuid.UUID
		meta := quizGamePeerMeta{Role: role}

		// IQ.9: guest players authenticate with playerToken only (no JWT).
		if role == "player" && m.PlayerToken != "" && m.AuthToken == "" {
			p, perr := quizgame.GetPlayerByToken(r.Context(), d.Pool, sess.ID, m.PlayerToken)
			if perr != nil || p == nil || p.Banned {
				return
			}
			if p.UserID != nil {
				// Enrolled players must present a JWT.
				return
			}
			if !sess.AllowGuests || !d.effectiveConfig().FFIqGuestJoin {
				return
			}
			meta.PlayerID = p.ID
			_ = quizgame.SetPlayerConnected(r.Context(), d.Pool, p.ID, true)
		} else {
			if m.AuthToken == "" {
				return
			}
			u, verr := d.JWTSigner.Verify(r.Context(), m.AuthToken)
			if verr != nil {
				return
			}
			parsed, perr := uuid.Parse(u.UserID)
			if perr != nil {
				return
			}
			userID = parsed
			has, herr := enrollment.UserHasAccess(r.Context(), d.Pool, courseCode, userID)
			if herr != nil || !has {
				return
			}
			switch role {
			case "host":
				hasPerm, perr := courseroles.UserHasPermission(r.Context(), d.Pool, userID, "course:"+courseCode+":item:create")
				if perr != nil || !hasPerm {
					return
				}
				// Host reconnect clears waiting state.
				if sess.CurrentPhase == string(engine.PhaseWaitingForHost) || sess.Status == string(engine.StatusPaused) {
					st := sess.EngineState()
					next, ev, ok := engine.ReduceHostReconnect(st, sess.HostDisconnectedAt, time.Now().UTC(), engine.HostGraceDefault)
					if ok {
						_, _ = quizgame.ApplyHostTransition(r.Context(), d.Pool, sess.ID, next, ev, false)
						sess, _ = quizgame.GetSession(r.Context(), d.Pool, sess.ID)
					}
				}
			case "projector":
				// Read-only; any enrolled user with course access may open projector for the host.
			case "player":
				if m.PlayerToken == "" {
					return
				}
				p, perr := quizgame.GetPlayerByToken(r.Context(), d.Pool, sess.ID, m.PlayerToken)
				if perr != nil || p == nil || p.Banned {
					return
				}
				meta.PlayerID = p.ID
				_ = quizgame.SetPlayerConnected(r.Context(), d.Pool, p.ID, true)
			default:
				return
			}
		}

		meta.GameID = gameID.String()
		client := &yrelay.Client{
			ID:     uuid.New(),
			UserID: userID,
			Conn:   conn,
		}
		setQuizGamePeer(client.ID, meta)
		room := globalQuizGameRooms.GetOrCreate(gameID)
		room.Add(client)
		telemetry.RecordBusinessEvent("quizgame.ws.join")
		defer func() {
			room.Remove(client.ID)
			clearQuizGamePeer(client.ID)
			globalQuizGameRooms.MaybeDelete(gameID)
			if meta.Role == "player" && meta.PlayerID != "" {
				_ = quizgame.SetPlayerConnected(context.Background(), d.Pool, meta.PlayerID, false)
			}
			if meta.Role == "host" {
				handleQuizHostDisconnect(context.Background(), d, gameID.String())
			}
			telemetry.RecordBusinessEvent("quizgame.ws.leave")
		}()

		// Snapshot state to the joining peer.
		if sess != nil {
			frame := buildQuizStateFrame(r.Context(), d, sess, meta.Role)
			writeCtx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
			_ = client.SendText(writeCtx, frame)
			cancel()
			if meta.Role == "player" && meta.PlayerID != "" {
				sendPlayerAnswerResume(r.Context(), d, client, sess, meta.PlayerID)
			}
		}

		runCtx, stop := context.WithCancel(r.Context())
		defer stop()

		var msgWindowStart time.Time
		var msgCount int

		for {
			msgType, data, err := conn.Read(runCtx)
			if err != nil {
				return
			}
			if msgType != websocket.MessageText || len(data) == 0 {
				continue
			}
			if len(data) > quizGameWSMaxMessageBytes {
				telemetry.RecordBusinessEvent("quizgame.ws.oversized")
				_ = conn.Close(websocket.StatusPolicyViolation, "message too large")
				return
			}
			now := time.Now()
			if msgWindowStart.IsZero() || now.Sub(msgWindowStart) >= time.Second {
				msgWindowStart = now
				msgCount = 0
			}
			msgCount++
			if msgCount > quizGameWSMaxMsgsPerSec && msgCount > quizGameWSBurst/2 {
				telemetry.RecordBusinessEvent("quizgame.ws.rate_limited")
				_ = conn.Close(websocket.StatusPolicyViolation, "rate limit exceeded")
				return
			}

			var frame struct {
				Type          string          `json:"type"`
				QuestionIndex *int            `json:"questionIndex"`
				Answer        json.RawMessage `json:"answer"`
				PlayerID      string          `json:"playerId"`
				Nickname      string          `json:"nickname"`
				Muted         *bool           `json:"muted"`
				Locked        *bool           `json:"locked"`
				AfterSeq      *int            `json:"afterSeq"`
				ResumeSeq     *int            `json:"resumeSeq"`
				ClientSentAt  string          `json:"clientSentAt"` // telemetry only; server clock is authoritative
				Kind          string          `json:"kind"`         // powerup kind
				PowerUp       string          `json:"powerUp"`      // answer-time power-up claim
			}
			if err := json.Unmarshal(data, &frame); err != nil || frame.Type == "" {
				continue
			}

			switch frame.Type {
			case "ping":
				writeCtx, cancel := context.WithTimeout(runCtx, 2*time.Second)
				_ = client.SendText(writeCtx, []byte(`{"type":"pong"}`))
				cancel()
			case "catchup", "hello":
				// IQ.3 catchup + IQ.4 hello alias (seq resync).
				after := 0
				if frame.AfterSeq != nil {
					after = *frame.AfterSeq
				} else if frame.ResumeSeq != nil {
					after = *frame.ResumeSeq
				}
				live, _ := quizgame.GetSession(runCtx, d.Pool, gameID.String())
				if live == nil {
					continue
				}
				stateFrame := buildQuizStateFrame(runCtx, d, live, meta.Role)
				writeCtx, cancel := context.WithTimeout(runCtx, 5*time.Second)
				_ = client.SendText(writeCtx, stateFrame)
				cancel()
				if meta.Role == "player" && meta.PlayerID != "" {
					sendPlayerAnswerResume(runCtx, d, client, live, meta.PlayerID)
				}
				events, _ := quizgame.ListEventsAfter(runCtx, d.Pool, gameID.String(), after)
				for _, ev := range events {
					payload, _ := json.Marshal(map[string]any{
						"type":    "event",
						"seq":     ev.Seq,
						"event":   ev.Type,
						"payload": json.RawMessage(ev.Payload),
					})
					writeCtx, cancel := context.WithTimeout(runCtx, 2*time.Second)
					_ = client.SendText(writeCtx, payload)
					cancel()
				}
			case "resume":
				// Host action resumes a paused game. Clients use catchup/hello for seq resync.
				// Legacy clients that sent {type:"resume", afterSeq} are treated as catchup.
				if frame.AfterSeq != nil || frame.ResumeSeq != nil {
					after := 0
					if frame.AfterSeq != nil {
						after = *frame.AfterSeq
					} else if frame.ResumeSeq != nil {
						after = *frame.ResumeSeq
					}
					live, _ := quizgame.GetSession(runCtx, d.Pool, gameID.String())
					if live == nil {
						continue
					}
					stateFrame := buildQuizStateFrame(runCtx, d, live, meta.Role)
					writeCtx, cancel := context.WithTimeout(runCtx, 5*time.Second)
					_ = client.SendText(writeCtx, stateFrame)
					cancel()
					if meta.Role == "player" && meta.PlayerID != "" {
						sendPlayerAnswerResume(runCtx, d, client, live, meta.PlayerID)
					}
					events, _ := quizgame.ListEventsAfter(runCtx, d.Pool, gameID.String(), after)
					for _, ev := range events {
						payload, _ := json.Marshal(map[string]any{
							"type":    "event",
							"seq":     ev.Seq,
							"event":   ev.Type,
							"payload": json.RawMessage(ev.Payload),
						})
						writeCtx, cancel := context.WithTimeout(runCtx, 2*time.Second)
						_ = client.SendText(writeCtx, payload)
						cancel()
					}
					continue
				}
				if meta.Role != "host" {
					continue
				}
				applyQuizHostAction(runCtx, d, gameID.String(), engine.ActionResume)
			case "open", "lock", "reveal", "next", "skip", "pause", "end":
				if meta.Role != "host" {
					continue
				}
				applyQuizHostAction(runCtx, d, gameID.String(), engine.HostAction(frame.Type))
			case "kick", "ban":
				if meta.Role != "host" || frame.PlayerID == "" {
					continue
				}
				_ = quizgame.BanPlayer(runCtx, d.Pool, gameID.String(), frame.PlayerID)
				pid, _ := uuid.Parse(frame.PlayerID)
				kind := quizgame.SafetyKicked
				if frame.Type == "ban" {
					kind = quizgame.SafetyBanned
				}
				_ = quizgame.RecordSafetyEvent(runCtx, d.Pool, gameID.String(), &pid, &userID, kind, map[string]any{"via": "ws"})
				notifyQuizPlayerKicked(runCtx, gameID, frame.PlayerID)
				live, _ := quizgame.GetSession(runCtx, d.Pool, gameID.String())
				if live != nil {
					broadcastQuizGameState(runCtx, d, live)
				}
				telemetry.RecordBusinessEvent("quizgame.player." + frame.Type)
			case "rename":
				if meta.Role != "host" || frame.PlayerID == "" {
					continue
				}
				nick := strings.TrimSpace(frame.Nickname)
				if nick == "" {
					nick, _ = quizgame.NeutralPlayerName(runCtx, d.Pool, gameID.String(), frame.PlayerID)
				}
				if err := quizgame.RenamePlayer(runCtx, d.Pool, gameID.String(), frame.PlayerID, nick); err != nil {
					continue
				}
				pid, _ := uuid.Parse(frame.PlayerID)
				_ = quizgame.RecordSafetyEvent(runCtx, d.Pool, gameID.String(), &pid, &userID, quizgame.SafetyRenamed, map[string]any{"nickname": nick, "via": "ws"})
				live, _ := quizgame.GetSession(runCtx, d.Pool, gameID.String())
				if live != nil {
					broadcastQuizGameState(runCtx, d, live)
				}
				telemetry.RecordBusinessEvent("quizgame.player.rename")
			case "mute_names":
				if meta.Role != "host" || frame.Muted == nil {
					continue
				}
				_ = quizgame.SetNamesMuted(runCtx, d.Pool, gameID.String(), *frame.Muted)
				_ = quizgame.RecordSafetyEvent(runCtx, d.Pool, gameID.String(), nil, &userID, quizgame.SafetyMuted, map[string]any{"muted": *frame.Muted, "via": "ws"})
				live, _ := quizgame.GetSession(runCtx, d.Pool, gameID.String())
				if live != nil {
					broadcastQuizGameState(runCtx, d, live)
				}
				telemetry.RecordBusinessEvent("quizgame.safety.mute_names")
			case "lock_lobby":
				if meta.Role != "host" || frame.Locked == nil {
					continue
				}
				_ = quizgame.SetLobbyLocked(runCtx, d.Pool, gameID.String(), *frame.Locked)
				_ = quizgame.RecordSafetyEvent(runCtx, d.Pool, gameID.String(), nil, &userID, quizgame.SafetyLobbyLocked, map[string]any{"locked": *frame.Locked, "via": "ws"})
				live, _ := quizgame.GetSession(runCtx, d.Pool, gameID.String())
				if live != nil {
					broadcastQuizGameState(runCtx, d, live)
				}
				telemetry.RecordBusinessEvent("quizgame.safety.lock_lobby")
			case "powerup":
				if meta.Role != "player" || meta.PlayerID == "" || frame.QuestionIndex == nil || frame.Kind == "" {
					continue
				}
				ack := map[string]any{
					"type":          "powerup_ack",
					"questionIndex": *frame.QuestionIndex,
					"kind":          frame.Kind,
				}
				if err := quizgame.ClaimPowerUp(runCtx, d.Pool, gameID.String(), meta.PlayerID, *frame.QuestionIndex, frame.Kind); err != nil {
					ack["ok"] = false
					switch {
					case errors.Is(err, quizgame.ErrPowerUpsDisabled):
						ack["error"] = "disabled"
					case errors.Is(err, quizgame.ErrPowerUpInvalid):
						ack["error"] = "ineligible"
					default:
						ack["error"] = "rejected"
					}
				} else {
					ack["ok"] = true
					telemetry.RecordBusinessEvent("quizgame.powerup.claim")
				}
				payload, _ := json.Marshal(ack)
				writeCtx, cancel := context.WithTimeout(runCtx, 2*time.Second)
				_ = client.SendText(writeCtx, payload)
				cancel()
			case "answer":
				if meta.Role != "player" || meta.PlayerID == "" || frame.QuestionIndex == nil {
					continue
				}
				_ = frame.ClientSentAt // telemetry hook; scoring uses server receipt time
				res, err := quizgame.SubmitAnswer(runCtx, d.Pool, quizgame.SubmitAnswerInput{
					SessionID:     gameID.String(),
					PlayerID:      meta.PlayerID,
					QuestionIndex: *frame.QuestionIndex,
					Answer:        frame.Answer,
					ReceivedAt:    time.Now().UTC(),
					PowerUp:       frame.PowerUp,
				})
				ack := map[string]any{
					"type":          "answer_ack",
					"questionIndex": *frame.QuestionIndex,
				}
				if err != nil {
					ack["ok"] = false
					if res != nil {
						ack["duplicate"] = res.Duplicate
						ack["late"] = res.Late
						ack["responseMs"] = res.ResponseMs
					}
					switch {
					case errors.Is(err, quizgame.ErrDuplicateAnswer):
						ack["error"] = "duplicate"
						telemetry.RecordBusinessEvent("quizgame.answer.dup")
						// Resume prior result so reconnecting clients lock UI.
						if prior, perr := quizgame.GetPlayerResponse(runCtx, d.Pool, gameID.String(), meta.PlayerID, *frame.QuestionIndex); perr == nil && prior != nil {
							ack["ok"] = true
							ack["isCorrect"] = prior.IsCorrect
							ack["points"] = prior.Points
							ack["responseMs"] = prior.ResponseMs
							ack["alreadyAnswered"] = true
							if len(prior.PointsBreakdown) > 0 {
								ack["pointsBreakdown"] = json.RawMessage(prior.PointsBreakdown)
							}
						}
					case errors.Is(err, quizgame.ErrLateAnswer):
						ack["error"] = "late"
						telemetry.RecordBusinessEvent("quizgame.answer.late")
					default:
						ack["error"] = "rejected"
					}
				} else {
					ack["ok"] = true
					ack["isCorrect"] = res.IsCorrect
					ack["responseMs"] = res.ResponseMs
					ack["points"] = res.Points
					ack["pointsBreakdown"] = res.PointsBreakdown
					telemetry.RecordBusinessEvent("quizgame.answer.ok")
					live, _ := quizgame.GetSession(runCtx, d.Pool, gameID.String())
					if live != nil {
						broadcastQuizGameState(runCtx, d, live)
					}
				}
				if p, perr := quizgame.GetPlayer(runCtx, d.Pool, meta.PlayerID); perr == nil && p != nil {
					ack["streak"] = p.Streak
					ack["totalScore"] = p.TotalScore
				}
				if rank, rerr := quizgame.PlayerRank(runCtx, d.Pool, gameID.String(), meta.PlayerID); rerr == nil {
					ack["rank"] = rank
				}
				payload, _ := json.Marshal(ack)
				writeCtx, cancel := context.WithTimeout(runCtx, 2*time.Second)
				_ = client.SendText(writeCtx, payload)
				cancel()
			}
		}
	}
}

func notifyQuizPlayerKicked(ctx context.Context, gameID uuid.UUID, playerID string) {
	room := globalQuizGameRooms.GetOrCreate(gameID)
	payload, _ := json.Marshal(map[string]any{
		"type":     "kicked",
		"playerId": playerID,
		"reason":   "host",
	})
	writeCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	room.ForEach(func(c *yrelay.Client) {
		meta, ok := getQuizGamePeer(c.ID)
		if !ok || meta.Role != "player" || meta.PlayerID != playerID {
			return
		}
		_ = c.SendText(writeCtx, payload)
		_ = c.Conn.Close(websocket.StatusNormalClosure, "kicked")
	})
}

func sendPlayerAnswerResume(ctx context.Context, d Deps, client *yrelay.Client, sess *quizgame.Session, playerID string) {
	if sess == nil || sess.CurrentIndex < 0 {
		return
	}
	phase := sess.CurrentPhase
	if phase != string(engine.PhaseQuestionOpen) &&
		phase != string(engine.PhaseQuestionLocked) &&
		phase != string(engine.PhaseQuestionReveal) {
		return
	}
	prior, err := quizgame.GetPlayerResponse(ctx, d.Pool, sess.ID, playerID, sess.CurrentIndex)
	if err != nil || prior == nil {
		return
	}
	ack := map[string]any{
		"type":            "answer_ack",
		"ok":              true,
		"questionIndex":   prior.QuestionIndex,
		"isCorrect":       prior.IsCorrect,
		"points":          prior.Points,
		"responseMs":      prior.ResponseMs,
		"alreadyAnswered": true,
	}
	if len(prior.PointsBreakdown) > 0 {
		ack["pointsBreakdown"] = json.RawMessage(prior.PointsBreakdown)
	}
	if p, perr := quizgame.GetPlayer(ctx, d.Pool, playerID); perr == nil && p != nil {
		ack["streak"] = p.Streak
		ack["totalScore"] = p.TotalScore
	}
	if rank, rerr := quizgame.PlayerRank(ctx, d.Pool, sess.ID, playerID); rerr == nil {
		ack["rank"] = rank
	}
	payload, _ := json.Marshal(ack)
	writeCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	_ = client.SendText(writeCtx, payload)
	cancel()
}

func handleQuizHostDisconnect(ctx context.Context, d Deps, sessionID string) {
	sess, err := quizgame.GetSession(ctx, d.Pool, sessionID)
	if err != nil || sess == nil {
		return
	}
	if sess.Status == "ended" || sess.Status == "abandoned" {
		return
	}
	// If another host tab is still connected, skip pause.
	id, err := uuid.Parse(sessionID)
	if err != nil {
		return
	}
	room := globalQuizGameRooms.GetOrCreate(id)
	hostStillThere := false
	// We don't track roles in yrelay clients; pause optimistically after short grace via reaper.
	// Immediate soft-pause for UX:
	_ = hostStillThere
	_ = room
	st := sess.EngineState()
	next, ev := engine.ReduceHostDisconnect(st, time.Now().UTC())
	_, _ = quizgame.ApplyHostTransition(ctx, d.Pool, sessionID, next, ev, true)
	live, _ := quizgame.GetSession(ctx, d.Pool, sessionID)
	if live != nil {
		broadcastQuizGameState(ctx, d, live)
	}
}

func applyQuizHostAction(ctx context.Context, d Deps, sessionID string, action engine.HostAction) {
	sess, err := quizgame.GetSession(ctx, d.Pool, sessionID)
	if err != nil || sess == nil {
		return
	}
	now := time.Now().UTC()
	st := sess.EngineState()
	next, ev, err := engine.Reduce(st, action, now)
	if err != nil {
		return
	}
	if next.Phase == engine.PhaseQuestionOpen && next.OpenedAt != nil {
		idx := next.QuestionIndex
		if idx >= 0 && idx < len(sess.KitSnapshot.Questions) {
			limit := sess.KitSnapshot.Questions[idx].TimeLimitSeconds
			next = engine.ApplyDeadline(next, limit)
			if len(ev) > 0 && ev[0].Type == "question_open" {
				if next.Deadline != nil {
					ev[0].Payload["deadline"] = next.Deadline.UTC().Format(time.RFC3339Nano)
				}
			}
		}
		// Schedule auto-lock at deadline.
		if next.Deadline != nil {
			deadline := *next.Deadline
			go func(sid string, dl time.Time) {
				timer := time.NewTimer(time.Until(dl))
				defer timer.Stop()
				<-timer.C
				autoLockQuizQuestion(context.Background(), d, sid, dl)
			}(sessionID, deadline)
		}
	}
	hostDisc := next.Phase == engine.PhaseWaitingForHost
	_, _ = quizgame.ApplyHostTransition(ctx, d.Pool, sessionID, next, ev, hostDisc)
	live, _ := quizgame.GetSession(ctx, d.Pool, sessionID)
	if live != nil {
		broadcastQuizGameState(ctx, d, live)
	}
}

func autoLockQuizQuestion(ctx context.Context, d Deps, sessionID string, expectedDeadline time.Time) {
	sess, err := quizgame.GetSession(ctx, d.Pool, sessionID)
	if err != nil || sess == nil {
		return
	}
	if sess.CurrentPhase != string(engine.PhaseQuestionOpen) {
		return
	}
	if sess.QuestionDeadlineAt == nil || !sess.QuestionDeadlineAt.Equal(expectedDeadline) {
		return
	}
	st := sess.EngineState()
	next, ev, err := engine.ReduceDeadlineLock(st, time.Now().UTC())
	if err != nil {
		return
	}
	_, _ = quizgame.ApplyHostTransition(ctx, d.Pool, sessionID, next, ev, false)
	live, _ := quizgame.GetSession(ctx, d.Pool, sessionID)
	if live != nil {
		broadcastQuizGameState(ctx, d, live)
	}
	// Auto pacing: auto-reveal → leaderboard → next after short delays.
	if live != nil && live.Pacing == string(engine.PacingAuto) {
		time.Sleep(2 * time.Second)
		applyQuizHostAction(ctx, d, sessionID, engine.ActionReveal)
		time.Sleep(4 * time.Second)
		applyQuizHostAction(ctx, d, sessionID, engine.ActionNext) // reveal → leaderboard
		time.Sleep(3 * time.Second)
		applyQuizHostAction(ctx, d, sessionID, engine.ActionNext) // leaderboard → next / podium
	}
}

func broadcastQuizGameState(ctx context.Context, d Deps, sess *quizgame.Session) {
	if sess == nil {
		return
	}
	id, err := uuid.Parse(sess.ID)
	if err != nil {
		return
	}
	room := globalQuizGameRooms.GetOrCreate(id)
	// Build role-specific frames: host/projector share structure but projector never gets correctness early.
	hostFrame := buildQuizStateFrame(ctx, d, sess, "host")
	projFrame := buildQuizStateFrame(ctx, d, sess, "projector")
	playerFrame := buildQuizStateFrame(ctx, d, sess, "player")
	writeCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	// Without per-client role storage on yrelay.Client, send the safe player/projector frame to all,
	// and also send host frame — hosts re-request via resume if needed. Prefer projector-safe broadcast.
	_ = hostFrame
	_ = playerFrame
	room.ForEach(func(c *yrelay.Client) {
		_ = c.SendText(writeCtx, projFrame)
	})
	// Hosts need correctness on reveal; send host frame as a second pass (harmless for players after reveal).
	if sess.CurrentPhase == string(engine.PhaseQuestionReveal) ||
		sess.CurrentPhase == string(engine.PhaseLeaderboard) ||
		sess.CurrentPhase == string(engine.PhasePodium) ||
		sess.CurrentPhase == string(engine.PhaseEnded) ||
		sess.CurrentPhase == string(engine.PhaseLobby) {
		room.ForEach(func(c *yrelay.Client) {
			meta, ok := getQuizGamePeer(c.ID)
			if ok && meta.Role == "host" {
				_ = c.SendText(writeCtx, hostFrame)
				return
			}
			if ok && meta.Role == "player" && meta.PlayerID != "" {
				_ = c.SendText(writeCtx, buildQuizStateFrameForPlayer(ctx, d, sess, meta.PlayerID))
				return
			}
			_ = c.SendText(writeCtx, projFrame)
		})
	}
}

func buildQuizStateFrameForPlayer(ctx context.Context, d Deps, sess *quizgame.Session, playerID string) []byte {
	b := buildQuizStateFrame(ctx, d, sess, "player")
	var frame map[string]any
	if json.Unmarshal(b, &frame) != nil {
		return b
	}
	view, err := quizgame.BuildLeaderboardView(ctx, d.Pool, sess, 10, playerID)
	if err == nil && view != nil && view.You != nil {
		frame["you"] = view.You
	}
	out, _ := json.Marshal(frame)
	return out
}

func buildQuizStateFrame(ctx context.Context, d Deps, sess *quizgame.Session, role string) []byte {
	seq, _ := quizgame.LatestSeq(ctx, d.Pool, sess.ID)
	players, _ := quizgame.ListPlayers(ctx, d.Pool, sess.ID)
	mode := engine.NormalizeMode(sess.Mode)
	muteForPublic := sess.NamesMuted && role != "host"
	playerOut := make([]map[string]any, 0, len(players))
	for i, p := range players {
		nick := p.Nickname
		if muteForPublic {
			nick = quizgame.DisplayNickname(true, i, p.Nickname)
		}
		row := map[string]any{
			"id":         p.ID,
			"nickname":   nick,
			"totalScore": p.TotalScore,
			"streak":     p.Streak,
			"connected":  p.Connected,
			"teamId":     p.TeamID,
		}
		if role == "host" {
			row["renamedByHost"] = p.RenamedByHost
			row["isGuest"] = p.UserID == nil
		}
		if !engine.UsesSharedClock(mode) {
			row["currentIndex"] = p.CurrentIndex
			row["currentPhase"] = p.CurrentPhase
			if p.FinishedAt != nil {
				row["finished"] = true
			}
		}
		playerOut = append(playerOut, row)
	}
	code := ""
	if sess.JoinCode != nil {
		code = *sess.JoinCode
	}
	frame := map[string]any{
		"type":          "state",
		"seq":           seq,
		"gameId":        sess.ID,
		"mode":          string(mode),
		"phase":         sess.CurrentPhase,
		"status":        sess.Status,
		"questionIndex": sess.CurrentIndex,
		"joinCode":      code,
		"kitTitle":      sess.KitSnapshot.Title,
		"pacing":        sess.Pacing,
		"players":       playerOut,
		"questionCount": len(sess.KitSnapshot.Questions),
		"namesMuted":    sess.NamesMuted,
		"lobbyLocked":   sess.LobbyLocked,
		"allowGuests":   sess.AllowGuests,
	}
	if sess.QuestionOpenedAt != nil {
		frame["openedAt"] = sess.QuestionOpenedAt.UTC().Format(time.RFC3339Nano)
	}
	if sess.QuestionDeadlineAt != nil {
		frame["deadline"] = sess.QuestionDeadlineAt.UTC().Format(time.RFC3339Nano)
	}

	idx := sess.CurrentIndex
	includeCorrect := role == "host" ||
		(role == "projector" && (sess.CurrentPhase == string(engine.PhaseQuestionReveal) ||
			sess.CurrentPhase == string(engine.PhaseLeaderboard) ||
			sess.CurrentPhase == string(engine.PhasePodium))) ||
		(role == "player" && sess.CurrentPhase == string(engine.PhaseQuestionReveal))

	if idx >= 0 && idx < len(sess.KitSnapshot.Questions) {
		q := sess.KitSnapshot.Questions[idx]
		pub := engine.ToPublicQuestion(q, idx)
		if q.AnswerShuffle {
			opts := pub.Options
			rand.Shuffle(len(opts), func(i, j int) { opts[i], opts[j] = opts[j], opts[i] })
			pub.Options = opts
		}
		qm := map[string]any{
			"index":            pub.Index,
			"questionType":     pub.QuestionType,
			"prompt":           pub.Prompt,
			"promptMediaRef":  pub.PromptMediaRef,
			"promptMediaAlt":  pub.PromptMediaAlt,
			"options":          pub.Options,
			"timeLimitSeconds": pub.TimeLimitSeconds,
			"pointsStyle":      pub.PointsStyle,
		}
		if includeCorrect {
			correctIDs := []string{}
			for _, o := range q.Options {
				if o.IsCorrect {
					correctIDs = append(correctIDs, o.ID)
				}
			}
			qm["correctOptionIds"] = correctIDs
			qm["correctAnswer"] = q.CorrectAnswer
			qm["explanation"] = q.Explanation
		}
		frame["question"] = qm
	}

	if idx >= 0 {
		n, _ := quizgame.CountAnswersForQuestion(ctx, d.Pool, sess.ID, idx)
		frame["answerCount"] = n
		if sess.CurrentPhase == string(engine.PhaseQuestionLocked) ||
			sess.CurrentPhase == string(engine.PhaseQuestionReveal) ||
			sess.CurrentPhase == string(engine.PhaseLeaderboard) {
			dist, _ := quizgame.AnswerDistribution(ctx, d.Pool, sess.ID, idx)
			if role == "projector" || role == "player" {
				dist = quizgame.FilterDistributionForProjector(dist)
			}
			frame["distribution"] = dist
		}
	}

	cfg := scoring.ResolveConfig(sess.ScoringProfile, scoring.ParseConfigJSON(sess.ScoringConfig))
	frame["scoringProfile"] = sess.ScoringProfile
	frame["powerUpsEnabled"] = cfg.PowerUpsEnabled
	frame["leaderboardPrivacy"] = scoring.NormalizePrivacy(sess.LeaderboardPrivacy)

	if sess.CurrentPhase == string(engine.PhaseQuestionReveal) ||
		sess.CurrentPhase == string(engine.PhaseLeaderboard) ||
		sess.CurrentPhase == string(engine.PhasePodium) ||
		sess.CurrentPhase == string(engine.PhaseEnded) {
		view, _ := quizgame.BuildLeaderboardView(ctx, d.Pool, sess, 10, "")
		if view != nil {
			lb := make([]map[string]any, 0, len(view.Top))
			for i, p := range view.Top {
				nick := p.Nickname
				if muteForPublic {
					nick = quizgame.DisplayNickname(true, i, p.Nickname)
				}
				row := map[string]any{
					"rank":       p.Rank,
					"playerId":   p.PlayerID,
					"nickname":   nick,
					"totalScore": p.TotalScore,
				}
				if view.Privacy == "hidden" {
					row["nickname"] = ""
				}
				lb = append(lb, row)
			}
			frame["leaderboard"] = lb
			if sess.CurrentPhase == string(engine.PhasePodium) || sess.CurrentPhase == string(engine.PhaseEnded) {
				frame["podium"] = lb
			}
		}
		if mode == engine.ModeTeam {
			if board, err := quizgame.RefreshTeamScores(ctx, d.Pool, sess.ID); err == nil {
				frame["teamLeaderboard"] = board
			}
		}
	}
	if mode == engine.ModeStudentPaced && (role == "host" || role == "projector") {
		buckets, total, finished, _ := quizgame.PacedHostProgress(ctx, d.Pool, sess.ID)
		frame["pacedProgress"] = buckets
		frame["playerCount"] = total
		frame["finishedCount"] = finished
	}

	b, _ := json.Marshal(frame)
	return b
}
