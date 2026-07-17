package quizgame

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// UserQuizExport is the DSAR export slice for Live Quizzes (IQ.11 FR-6).
type UserQuizExport struct {
	Players   []map[string]any `json:"players"`
	Responses []map[string]any `json:"responses"`
	Results   []map[string]any `json:"results"`
}

// ExportUserContent collects a user's game participations, responses, and scores.
func ExportUserContent(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID) (UserQuizExport, error) {
	out := UserQuizExport{
		Players:   []map[string]any{},
		Responses: []map[string]any{},
		Results:   []map[string]any{},
	}

	prows, err := pool.Query(ctx, `
		SELECT p.id, p.session_id, c.course_code, p.nickname, p.total_score, p.streak,
		       p.joined_at, s.status, s.mode, s.ended_at
		FROM quizgame.session_players p
		INNER JOIN quizgame.sessions s ON s.id = p.session_id
		INNER JOIN course.courses c ON c.id = s.course_id
		WHERE p.user_id = $1 AND p.removed_at IS NULL
		ORDER BY p.joined_at ASC
	`, userID)
	if err != nil {
		return out, err
	}
	defer prows.Close()
	for prows.Next() {
		var playerID, sessionID uuid.UUID
		var courseCode, nickname, status, mode string
		var totalScore, streak int
		var joinedAt time.Time
		var endedAt *time.Time
		if err := prows.Scan(&playerID, &sessionID, &courseCode, &nickname, &totalScore, &streak,
			&joinedAt, &status, &mode, &endedAt); err != nil {
			return out, err
		}
		row := map[string]any{
			"playerId":   playerID.String(),
			"sessionId":  sessionID.String(),
			"courseCode": courseCode,
			"nickname":   nickname,
			"totalScore": totalScore,
			"streak":     streak,
			"joinedAt":   joinedAt.UTC().Format(time.RFC3339),
			"status":     status,
			"mode":       mode,
		}
		if endedAt != nil {
			row["endedAt"] = endedAt.UTC().Format(time.RFC3339)
		}
		out.Players = append(out.Players, row)
		out.Results = append(out.Results, map[string]any{
			"sessionId":  sessionID.String(),
			"courseCode": courseCode,
			"totalScore": totalScore,
			"streak":     streak,
			"mode":       mode,
			"status":     status,
		})
	}
	if err := prows.Err(); err != nil {
		return out, err
	}

	rrows, err := pool.Query(ctx, `
		SELECT r.session_id, r.question_index, r.player_id, r.answer, r.is_correct,
		       r.response_ms, r.points, r.answered_at, c.course_code
		FROM quizgame.session_responses r
		INNER JOIN quizgame.session_players p ON p.id = r.player_id
		INNER JOIN quizgame.sessions s ON s.id = r.session_id
		INNER JOIN course.courses c ON c.id = s.course_id
		WHERE p.user_id = $1
		ORDER BY r.answered_at ASC
	`, userID)
	if err != nil {
		return out, err
	}
	defer rrows.Close()
	for rrows.Next() {
		var sessionID, playerID uuid.UUID
		var qIndex, responseMs, points int
		var answer []byte
		var isCorrect bool
		var answeredAt time.Time
		var courseCode string
		if err := rrows.Scan(&sessionID, &qIndex, &playerID, &answer, &isCorrect,
			&responseMs, &points, &answeredAt, &courseCode); err != nil {
			return out, err
		}
		row := map[string]any{
			"sessionId":     sessionID.String(),
			"playerId":      playerID.String(),
			"courseCode":    courseCode,
			"questionIndex": qIndex,
			"isCorrect":     isCorrect,
			"responseMs":    responseMs,
			"points":        points,
			"answeredAt":    answeredAt.UTC().Format(time.RFC3339),
		}
		if len(answer) > 0 {
			var parsed any
			if json.Unmarshal(answer, &parsed) == nil {
				row["answer"] = parsed
			}
		}
		out.Responses = append(out.Responses, row)
	}
	return out, rrows.Err()
}

// EraseUserContent anonymises a user's quizgame participations (IQ.11 FR-6 / AC-7).
// Guest rows for the user do not exist (guests have null user_id); enrolled rows are anonymised.
func EraseUserContent(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID) error {
	if _, err := pool.Exec(ctx, `
		UPDATE quizgame.session_responses r
		SET answer = '{"redacted":true}'::jsonb
		FROM quizgame.session_players p
		WHERE r.player_id = p.id AND p.user_id = $1
	`, userID); err != nil {
		return err
	}
	if _, err := pool.Exec(ctx, `
		UPDATE quizgame.session_players
		SET nickname = 'Redacted',
		    user_id = NULL,
		    client_meta = '{}'::jsonb,
		    join_ip_hash = NULL
		WHERE user_id = $1
	`, userID); err != nil {
		return err
	}
	if _, err := pool.Exec(ctx, `
		UPDATE quizgame.kits
		SET created_by = NULL
		WHERE created_by = $1
	`, userID); err != nil {
		return err
	}
	if _, err := pool.Exec(ctx, `
		UPDATE quizgame.generation_jobs
		SET requested_by = NULL
		WHERE requested_by = $1
	`, userID); err != nil {
		return err
	}
	return nil
}

// CountUserContentRows returns remaining identifiable rows for verification after erasure.
func CountUserContentRows(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID) (int64, error) {
	var n int64
	err := pool.QueryRow(ctx, `
		SELECT
			(SELECT COUNT(*) FROM quizgame.session_players WHERE user_id = $1) +
			(SELECT COUNT(*) FROM quizgame.kits WHERE created_by = $1) +
			(SELECT COUNT(*) FROM quizgame.generation_jobs WHERE requested_by = $1) +
			(SELECT COUNT(*) FROM quizgame.session_responses r
			  INNER JOIN quizgame.session_players p ON p.id = r.player_id
			 WHERE p.user_id = $1 AND r.answer IS DISTINCT FROM '{"redacted":true}'::jsonb)
	`, userID).Scan(&n)
	return n, err
}
