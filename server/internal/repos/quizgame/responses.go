package quizgame

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lextures/lextures/server/internal/quizgame/engine"
	"github.com/lextures/lextures/server/internal/quizgame/scoring"
)

// Response is one scored answer.
type Response struct {
	SessionID       string
	QuestionIndex   int
	PlayerID        string
	Answer          json.RawMessage
	IsCorrect       bool
	ResponseMs      int
	Points          int
	PointsBreakdown json.RawMessage
	AnsweredAt      time.Time
}

// SubmitAnswerInput is a player answer attempt.
type SubmitAnswerInput struct {
	SessionID     string
	PlayerID      string
	QuestionIndex int
	Answer        json.RawMessage
	ReceivedAt    time.Time
	PowerUp       string // optional; validated server-side when power-ups enabled
}

// SubmitAnswerResult is the accepted or rejected outcome.
type SubmitAnswerResult struct {
	Accepted        bool
	Duplicate       bool
	Late            bool
	IsCorrect       bool
	ResponseMs      int
	Points          int
	PointsBreakdown scoring.Breakdown
	Streak          int
	TotalScore      int
}

// SubmitAnswer records an idempotent answer using the server clock and IQ.5 scoring.
func SubmitAnswer(ctx context.Context, pool *pgxpool.Pool, in SubmitAnswerInput) (*SubmitAnswerResult, error) {
	sess, err := GetSession(ctx, pool, in.SessionID)
	if err != nil {
		return nil, err
	}
	mode := engine.NormalizeMode(sess.Mode)
	player, err := GetPlayer(ctx, pool, in.PlayerID)
	if err != nil || player == nil {
		return nil, ErrPlayerNotFound
	}
	if player.SessionID != in.SessionID {
		return nil, ErrPlayerNotFound
	}

	var openedAt time.Time
	var deadline *time.Time
	kitQuestionIndex := in.QuestionIndex

	if engine.UsesSharedClock(mode) {
		if sess.CurrentPhase != string(engine.PhaseQuestionOpen) {
			return nil, ErrNotAccepting
		}
		if sess.CurrentIndex != in.QuestionIndex {
			return nil, ErrNotAccepting
		}
		if sess.QuestionOpenedAt == nil {
			return nil, ErrNotAccepting
		}
		openedAt = *sess.QuestionOpenedAt
		deadline = sess.QuestionDeadlineAt
	} else {
		// student_paced / homework: per-player clock
		if player.FinishedAt != nil || player.CurrentPhase != string(engine.PhaseQuestionOpen) {
			return nil, ErrNotAccepting
		}
		if engine.TimeBudgetExpired(player.TimeBudgetEndsAt, in.ReceivedAt) {
			_, _ = FinalizePlayerPaced(ctx, pool, in.PlayerID, in.ReceivedAt)
			return nil, ErrNotAccepting
		}
		if player.QuestionOpenedAt == nil {
			return nil, ErrNotAccepting
		}
		openedAt = *player.QuestionOpenedAt
		deadline = player.QuestionDeadlineAt
		// Client may send kit index or progress index; prefer kit index from order.
		if len(player.QuestionOrder) > 0 {
			resolved, ok := engine.ResolveQuestionIndex(player.QuestionOrder, player.CurrentIndex)
			if !ok {
				return nil, ErrNotAccepting
			}
			kitQuestionIndex = resolved
			if in.QuestionIndex != kitQuestionIndex && in.QuestionIndex != player.CurrentIndex {
				return nil, ErrNotAccepting
			}
		} else if player.CurrentIndex != in.QuestionIndex {
			return nil, ErrNotAccepting
		}
	}

	ms, late := engine.ResponseTiming(openedAt, deadline, in.ReceivedAt)
	if late {
		return &SubmitAnswerResult{Accepted: false, Late: true, ResponseMs: ms}, ErrLateAnswer
	}
	if kitQuestionIndex < 0 || kitQuestionIndex >= len(sess.KitSnapshot.Questions) {
		return nil, ErrNotAccepting
	}
	q := sess.KitSnapshot.Questions[kitQuestionIndex]
	correct := engine.GradeAnswer(q, in.Answer)

	// Team one_device_per_team: only one answer per team per question (AC-2).
	if mode == engine.ModeTeam && player.TeamID != nil {
		msSettings := engine.ParseModeSettings(sess.Settings)
		tc := engine.NormalizeTeamConfig(msSettings.Team)
		if tc.AnswerRule == engine.TeamAnswerOneDevice {
			already, aerr := TeamAlreadyAnswered(ctx, pool, in.SessionID, *player.TeamID, kitQuestionIndex)
			if aerr != nil {
				return nil, aerr
			}
			if already {
				return nil, ErrOneDeviceAnswered
			}
		}
	}

	cfg := scoring.ResolveConfig(sess.ScoringProfile, scoring.ParseConfigJSON(sess.ScoringConfig))
	timeLimitMs := q.TimeLimitSeconds * 1000
	if deadline != nil {
		timeLimitMs = int(deadline.Sub(openedAt) / time.Millisecond)
	}

	powerUp := ""
	shieldActive := false
	if cfg.PowerUpsEnabled && in.PowerUp != "" {
		ok, serr := ValidatePowerUpClaim(ctx, pool, sess, player, kitQuestionIndex, in.PowerUp)
		if serr != nil {
			return nil, serr
		}
		if ok {
			powerUp = in.PowerUp
			if powerUp == scoring.PowerUpShield {
				used, uerr := PlayerUsedPowerUpKind(ctx, pool, in.SessionID, in.PlayerID, scoring.PowerUpShield)
				if uerr != nil {
					return nil, uerr
				}
				shieldActive = !used
			}
		}
	}

	scored := scoring.Score(scoring.Input{
		IsCorrect:    correct,
		ResponseMs:   ms,
		TimeLimitMs:  timeLimitMs,
		PointsStyle:  q.PointsStyle,
		QuestionType: q.QuestionType,
		StreakBefore: player.Streak,
		Profile:      sess.ScoringProfile,
		Config:       cfg,
		PowerUp:      powerUp,
		ShieldActive: shieldActive,
	})
	bdJSON, _ := json.Marshal(scored.Breakdown)

	sid, err := uuid.Parse(in.SessionID)
	if err != nil {
		return nil, err
	}
	pid, err := uuid.Parse(in.PlayerID)
	if err != nil {
		return nil, ErrPlayerNotFound
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	_, err = tx.Exec(ctx, `
		INSERT INTO quizgame.session_responses
			(session_id, question_index, player_id, answer, is_correct, response_ms, points, points_breakdown, answered_at)
		VALUES ($1, $2, $3, $4::jsonb, $5, $6, $7, $8::jsonb, $9)`,
		sid, kitQuestionIndex, pid, []byte(in.Answer), correct, ms, scored.Points, bdJSON, in.ReceivedAt.UTC(),
	)
	if err != nil {
		if isUniqueViolation(err) {
			return &SubmitAnswerResult{Accepted: false, Duplicate: true}, ErrDuplicateAnswer
		}
		return nil, err
	}
	if err := SetPlayerScoreAndStreak(ctx, tx, in.PlayerID, scored.Points, scored.StreakAfter); err != nil {
		return nil, err
	}
	// Ledger only when the power-up actually applied (shield on miss; don always when claimed).
	if powerUp == scoring.PowerUpDoubleOrNothing || scored.ShieldUsed {
		if err := RecordPowerUp(ctx, tx, in.SessionID, in.PlayerID, kitQuestionIndex, powerUp); err != nil {
			return nil, err
		}
	}
	if _, err := AppendEvent(ctx, tx, in.SessionID, "answer", map[string]any{
		"playerId":        in.PlayerID,
		"questionIndex":   kitQuestionIndex,
		"isCorrect":       correct,
		"responseMs":      ms,
		"points":          scored.Points,
		"pointsBreakdown": scored.Breakdown,
		"powerUp":         powerUp,
	}); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	if mode == engine.ModeTeam {
		_, _ = RefreshTeamScores(ctx, pool, in.SessionID)
	}
	if !engine.UsesSharedClock(mode) {
		_, _ = AdvancePlayerPaced(ctx, pool, in.PlayerID, in.ReceivedAt)
	}
	total := player.TotalScore + scored.Points
	return &SubmitAnswerResult{
		Accepted:        true,
		IsCorrect:       correct,
		ResponseMs:      ms,
		Points:          scored.Points,
		PointsBreakdown: scored.Breakdown,
		Streak:          scored.StreakAfter,
		TotalScore:      total,
	}, nil
}

// CountAnswersForQuestion returns how many players answered the current question.
func CountAnswersForQuestion(ctx context.Context, pool *pgxpool.Pool, sessionID string, questionIndex int) (int, error) {
	sid, err := uuid.Parse(sessionID)
	if err != nil {
		return 0, err
	}
	var n int
	err = pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM quizgame.session_responses
		WHERE session_id = $1 AND question_index = $2`, sid, questionIndex).Scan(&n)
	return n, err
}

// AnswerDistribution aggregates option picks for projector charts (reveal-safe counts only).
func AnswerDistribution(ctx context.Context, pool *pgxpool.Pool, sessionID string, questionIndex int) (map[string]int, error) {
	sid, err := uuid.Parse(sessionID)
	if err != nil {
		return nil, err
	}
	rows, err := pool.Query(ctx, `
		SELECT answer FROM quizgame.session_responses
		WHERE session_id = $1 AND question_index = $2`, sid, questionIndex)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]int{}
	for rows.Next() {
		var raw []byte
		if err := rows.Scan(&raw); err != nil {
			return nil, err
		}
		var body struct {
			OptionID  string   `json:"optionId"`
			OptionIDs []string `json:"optionIds"`
			Text      string   `json:"text"`
		}
		if json.Unmarshal(raw, &body) != nil {
			continue
		}
		if body.OptionID != "" {
			out[body.OptionID]++
		}
		for _, id := range body.OptionIDs {
			out[id]++
		}
		if body.Text != "" {
			out[body.Text]++
		}
	}
	return out, rows.Err()
}

// FilterDistributionForProjector removes open-text keys that fail content moderation (fail-closed).
func FilterDistributionForProjector(dist map[string]int) map[string]int {
	if len(dist) == 0 {
		return dist
	}
	out := make(map[string]int, len(dist))
	for k, v := range dist {
		// Option IDs are short opaque tokens (uuid-ish / single letters); free-text can be longer.
		// Screen every key — option ids won't match the blocklist.
		if denied, _ := ScreenOpenText(k); denied {
			continue
		}
		out[k] = v
	}
	return out
}

// GetPlayerResponse returns one player's response for a question, if any.
func GetPlayerResponse(ctx context.Context, pool *pgxpool.Pool, sessionID, playerID string, questionIndex int) (*Response, error) {
	sid, err := uuid.Parse(sessionID)
	if err != nil {
		return nil, ErrSessionNotFound
	}
	pid, err := uuid.Parse(playerID)
	if err != nil {
		return nil, ErrPlayerNotFound
	}
	var r Response
	var sidU, pidU uuid.UUID
	var answer, breakdown []byte
	err = pool.QueryRow(ctx, `
		SELECT session_id, question_index, player_id, answer, is_correct, response_ms, points, points_breakdown, answered_at
		FROM quizgame.session_responses
		WHERE session_id = $1 AND player_id = $2 AND question_index = $3`,
		sid, pid, questionIndex,
	).Scan(&sidU, &r.QuestionIndex, &pidU, &answer, &r.IsCorrect, &r.ResponseMs, &r.Points, &breakdown, &r.AnsweredAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	r.SessionID = sidU.String()
	r.PlayerID = pidU.String()
	r.Answer = answer
	r.PointsBreakdown = breakdown
	return &r, nil
}

// ListResponsesForQuestion returns all responses for scoring/reveal.
func ListResponsesForQuestion(ctx context.Context, pool *pgxpool.Pool, sessionID string, questionIndex int) ([]Response, error) {
	sid, err := uuid.Parse(sessionID)
	if err != nil {
		return nil, err
	}
	rows, err := pool.Query(ctx, `
		SELECT session_id, question_index, player_id, answer, is_correct, response_ms, points, points_breakdown, answered_at
		FROM quizgame.session_responses
		WHERE session_id = $1 AND question_index = $2`, sid, questionIndex)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Response
	for rows.Next() {
		var r Response
		var sidU, pid uuid.UUID
		var answer, breakdown []byte
		if err := rows.Scan(&sidU, &r.QuestionIndex, &pid, &answer, &r.IsCorrect, &r.ResponseMs, &r.Points, &breakdown, &r.AnsweredAt); err != nil {
			return nil, err
		}
		r.SessionID = sidU.String()
		r.PlayerID = pid.String()
		r.Answer = answer
		r.PointsBreakdown = breakdown
		out = append(out, r)
	}
	return out, rows.Err()
}

// ListPlayerResponses returns all responses for a player with breakdowns (IQ.7 feed).
func ListPlayerResponses(ctx context.Context, pool *pgxpool.Pool, sessionID, playerID string) ([]Response, error) {
	sid, err := uuid.Parse(sessionID)
	if err != nil {
		return nil, ErrSessionNotFound
	}
	pid, err := uuid.Parse(playerID)
	if err != nil {
		return nil, ErrPlayerNotFound
	}
	rows, err := pool.Query(ctx, `
		SELECT session_id, question_index, player_id, answer, is_correct, response_ms, points, points_breakdown, answered_at
		FROM quizgame.session_responses
		WHERE session_id = $1 AND player_id = $2
		ORDER BY question_index ASC`, sid, pid)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Response
	for rows.Next() {
		var r Response
		var sidU, pidU uuid.UUID
		var answer, breakdown []byte
		if err := rows.Scan(&sidU, &r.QuestionIndex, &pidU, &answer, &r.IsCorrect, &r.ResponseMs, &r.Points, &breakdown, &r.AnsweredAt); err != nil {
			return nil, err
		}
		r.SessionID = sidU.String()
		r.PlayerID = pidU.String()
		r.Answer = answer
		r.PointsBreakdown = breakdown
		out = append(out, r)
	}
	return out, rows.Err()
}

// ApplyHostTransition persists a host-driven state change + events atomically.
func ApplyHostTransition(ctx context.Context, pool *pgxpool.Pool, sessionID string, next engine.State, events []engine.Event, hostDisconnected bool) (int, error) {
	clearCode := next.Status == engine.StatusEnded || next.Status == engine.StatusAbandoned || next.Phase == engine.PhaseEnded
	tx, err := pool.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := PersistStateFull(ctx, tx, sessionID, next, clearCode, hostDisconnected); err != nil {
		return 0, err
	}
	var lastSeq int
	for _, ev := range events {
		seq, err := AppendEvent(ctx, tx, sessionID, ev.Type, ev.Payload)
		if err != nil {
			return 0, err
		}
		lastSeq = seq
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, err
	}
	return lastSeq, nil
}

// EndSession forces end from host REST.
func EndSession(ctx context.Context, pool *pgxpool.Pool, sessionID string, now time.Time) (*Session, error) {
	sess, err := GetSession(ctx, pool, sessionID)
	if err != nil {
		return nil, err
	}
	if sess.Status == "ended" || sess.Status == "abandoned" {
		return sess, nil
	}
	st := sess.EngineState()
	next, ev, err := engine.Reduce(st, engine.ActionEnd, now)
	if err != nil {
		// Force end even from unexpected phases.
		next = st
		next.Phase = engine.PhaseEnded
		next.Status = engine.StatusEnded
		ev = []engine.Event{{Type: "ended", Payload: map[string]any{"at": now.UTC().Format(time.RFC3339Nano)}}}
	}
	if _, err := ApplyHostTransition(ctx, pool, sessionID, next, ev, false); err != nil {
		return nil, err
	}
	return GetSession(ctx, pool, sessionID)
}
