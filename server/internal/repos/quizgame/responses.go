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
)

// Response is one scored answer.
type Response struct {
	SessionID     string
	QuestionIndex int
	PlayerID      string
	Answer        json.RawMessage
	IsCorrect     bool
	ResponseMs    int
	Points        int
	AnsweredAt    time.Time
}

// SubmitAnswerInput is a player answer attempt.
type SubmitAnswerInput struct {
	SessionID     string
	PlayerID      string
	QuestionIndex int
	Answer        json.RawMessage
	ReceivedAt    time.Time
}

// SubmitAnswerResult is the accepted or rejected outcome.
type SubmitAnswerResult struct {
	Accepted   bool
	Duplicate  bool
	Late       bool
	IsCorrect  bool
	ResponseMs int
	Points     int
}

// SubmitAnswer records an idempotent answer using the server clock (FR-6).
func SubmitAnswer(ctx context.Context, pool *pgxpool.Pool, in SubmitAnswerInput) (*SubmitAnswerResult, error) {
	sess, err := GetSession(ctx, pool, in.SessionID)
	if err != nil {
		return nil, err
	}
	if sess.CurrentPhase != string(engine.PhaseQuestionOpen) {
		return nil, ErrNotAccepting
	}
	if sess.CurrentIndex != in.QuestionIndex {
		return nil, ErrNotAccepting
	}
	if sess.QuestionOpenedAt == nil {
		return nil, ErrNotAccepting
	}
	ms, late := engine.ResponseTiming(*sess.QuestionOpenedAt, sess.QuestionDeadlineAt, in.ReceivedAt)
	if late {
		return &SubmitAnswerResult{Accepted: false, Late: true, ResponseMs: ms}, ErrLateAnswer
	}
	if in.QuestionIndex < 0 || in.QuestionIndex >= len(sess.KitSnapshot.Questions) {
		return nil, ErrNotAccepting
	}
	q := sess.KitSnapshot.Questions[in.QuestionIndex]
	correct := engine.GradeAnswer(q, in.Answer)
	points := engine.StubPoints(q.PointsStyle, correct, q.QuestionType)

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
			(session_id, question_index, player_id, answer, is_correct, response_ms, points, answered_at)
		VALUES ($1, $2, $3, $4::jsonb, $5, $6, $7, $8)`,
		sid, in.QuestionIndex, pid, []byte(in.Answer), correct, ms, points, in.ReceivedAt.UTC(),
	)
	if err != nil {
		if isUniqueViolation(err) {
			return &SubmitAnswerResult{Accepted: false, Duplicate: true}, ErrDuplicateAnswer
		}
		return nil, err
	}
	if err := AddPlayerScore(ctx, tx, in.PlayerID, points, correct); err != nil {
		return nil, err
	}
	if _, err := AppendEvent(ctx, tx, in.SessionID, "answer", map[string]any{
		"playerId":      in.PlayerID,
		"questionIndex": in.QuestionIndex,
		"isCorrect":     correct,
		"responseMs":    ms,
		"points":        points,
	}); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return &SubmitAnswerResult{
		Accepted:   true,
		IsCorrect:  correct,
		ResponseMs: ms,
		Points:     points,
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
	var answer []byte
	err = pool.QueryRow(ctx, `
		SELECT session_id, question_index, player_id, answer, is_correct, response_ms, points, answered_at
		FROM quizgame.session_responses
		WHERE session_id = $1 AND player_id = $2 AND question_index = $3`,
		sid, pid, questionIndex,
	).Scan(&sidU, &r.QuestionIndex, &pidU, &answer, &r.IsCorrect, &r.ResponseMs, &r.Points, &r.AnsweredAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	r.SessionID = sidU.String()
	r.PlayerID = pidU.String()
	r.Answer = answer
	return &r, nil
}

// PlayerRank returns 1-based rank by total_score (ties broken by joined_at).
func PlayerRank(ctx context.Context, pool *pgxpool.Pool, sessionID, playerID string) (int, error) {
	sid, err := uuid.Parse(sessionID)
	if err != nil {
		return 0, ErrSessionNotFound
	}
	pid, err := uuid.Parse(playerID)
	if err != nil {
		return 0, ErrPlayerNotFound
	}
	var rank int
	err = pool.QueryRow(ctx, `
		SELECT rank FROM (
			SELECT id, RANK() OVER (ORDER BY total_score DESC, joined_at ASC) AS rank
			FROM quizgame.session_players
			WHERE session_id = $1 AND removed_at IS NULL
		) t WHERE id = $2`, sid, pid).Scan(&rank)
	return rank, err
}

// ListResponsesForQuestion returns all responses for scoring/reveal.
func ListResponsesForQuestion(ctx context.Context, pool *pgxpool.Pool, sessionID string, questionIndex int) ([]Response, error) {
	sid, err := uuid.Parse(sessionID)
	if err != nil {
		return nil, err
	}
	rows, err := pool.Query(ctx, `
		SELECT session_id, question_index, player_id, answer, is_correct, response_ms, points, answered_at
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
		var answer []byte
		if err := rows.Scan(&sidU, &r.QuestionIndex, &pid, &answer, &r.IsCorrect, &r.ResponseMs, &r.Points, &r.AnsweredAt); err != nil {
			return nil, err
		}
		r.SessionID = sidU.String()
		r.PlayerID = pid.String()
		r.Answer = answer
		out = append(out, r)
	}
	return out, rows.Err()
}

// LeaderboardRows returns players ordered by score.
func LeaderboardRows(ctx context.Context, pool *pgxpool.Pool, sessionID string, limit int) ([]Player, error) {
	if limit <= 0 {
		limit = 10
	}
	sid, err := uuid.Parse(sessionID)
	if err != nil {
		return nil, err
	}
	rows, err := pool.Query(ctx, `
		SELECT id, session_id, user_id, nickname, total_score, streak, connected, joined_at, removed_at, last_seen_at
		FROM quizgame.session_players
		WHERE session_id = $1 AND removed_at IS NULL
		ORDER BY total_score DESC, joined_at ASC
		LIMIT $2`, sid, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Player
	for rows.Next() {
		p, err := scanPlayer(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *p)
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
