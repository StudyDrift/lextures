package quizgame

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lextures/lextures/server/internal/quizgame/scoring"
)

// LeaderboardEntry is one ranked row for fan-out / REST.
type LeaderboardEntry struct {
	Rank       int    `json:"rank"`
	PlayerID   string `json:"playerId"`
	Nickname   string `json:"nickname"`
	DisplayName string `json:"displayName,omitempty"` // legal name when privacy=names and available
	TotalScore int    `json:"totalScore"`
	Streak     int    `json:"streak"`
	Delta      *int   `json:"delta,omitempty"` // rank change vs prior snapshot (optional)
}

// LeaderboardYou is the requesting player's standing (FR-6).
type LeaderboardYou struct {
	Rank       int `json:"rank"`
	TotalScore int `json:"totalScore"`
	Streak     int `json:"streak"`
	Delta      *int `json:"delta,omitempty"`
}

// LeaderboardView is top-N + optional you payload.
type LeaderboardView struct {
	Top            []LeaderboardEntry `json:"top"`
	You            *LeaderboardYou    `json:"you,omitempty"`
	Privacy        string             `json:"privacy"`
	PlayerCount    int                `json:"playerCount"`
}

// ComputeLeaderboard returns players ordered by total_score DESC, then fewer
// total response_ms, then earliest join (FR-6 tie-break).
func ComputeLeaderboard(ctx context.Context, pool *pgxpool.Pool, sessionID string, limit int) ([]LeaderboardEntry, error) {
	if limit <= 0 {
		limit = 10
	}
	sid, err := uuid.Parse(sessionID)
	if err != nil {
		return nil, ErrSessionNotFound
	}
	rows, err := pool.Query(ctx, `
		SELECT p.id, p.nickname, p.total_score, p.streak,
			RANK() OVER (
				ORDER BY p.total_score DESC,
					COALESCE((
						SELECT SUM(r.response_ms) FROM quizgame.session_responses r
						WHERE r.session_id = p.session_id AND r.player_id = p.id
					), 0) ASC,
					p.joined_at ASC
			) AS rank
		FROM quizgame.session_players p
		WHERE p.session_id = $1 AND p.removed_at IS NULL
		ORDER BY rank ASC
		LIMIT $2`, sid, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []LeaderboardEntry
	for rows.Next() {
		var e LeaderboardEntry
		var id uuid.UUID
		if err := rows.Scan(&id, &e.Nickname, &e.TotalScore, &e.Streak, &e.Rank); err != nil {
			return nil, err
		}
		e.PlayerID = id.String()
		out = append(out, e)
	}
	return out, rows.Err()
}

// PlayerRank returns 1-based rank with IQ.5 tie-break (total response_ms, then join).
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
			SELECT p.id,
				RANK() OVER (
					ORDER BY p.total_score DESC,
						COALESCE((
							SELECT SUM(r.response_ms) FROM quizgame.session_responses r
							WHERE r.session_id = p.session_id AND r.player_id = p.id
						), 0) ASC,
						p.joined_at ASC
				) AS rank
			FROM quizgame.session_players p
			WHERE p.session_id = $1 AND p.removed_at IS NULL
		) t WHERE id = $2`, sid, pid).Scan(&rank)
	return rank, err
}

// BuildLeaderboardView applies privacy (FR-11 / AC-8) and optional you standing.
func BuildLeaderboardView(ctx context.Context, pool *pgxpool.Pool, sess *Session, topN int, viewerPlayerID string) (*LeaderboardView, error) {
	privacy := scoring.NormalizePrivacy(sess.LeaderboardPrivacy)
	top, err := ComputeLeaderboard(ctx, pool, sess.ID, topN)
	if err != nil {
		return nil, err
	}
	count, err := CountActivePlayers(ctx, pool, sess.ID)
	if err != nil {
		return nil, err
	}
	view := &LeaderboardView{
		Top:         applyLeaderboardPrivacy(top, privacy),
		Privacy:     privacy,
		PlayerCount: count,
	}
	if viewerPlayerID != "" {
		p, err := GetPlayer(ctx, pool, viewerPlayerID)
		if err == nil && p != nil && p.SessionID == sess.ID {
			rank, rerr := PlayerRank(ctx, pool, sess.ID, viewerPlayerID)
			if rerr == nil {
				view.You = &LeaderboardYou{
					Rank:       rank,
					TotalScore: p.TotalScore,
					Streak:     p.Streak,
				}
			}
		}
	}
	return view, nil
}

func applyLeaderboardPrivacy(rows []LeaderboardEntry, privacy string) []LeaderboardEntry {
	out := make([]LeaderboardEntry, len(rows))
	for i, r := range rows {
		e := r
		switch privacy {
		case scoring.PrivacyHidden:
			e.Nickname = ""
			e.DisplayName = ""
		case scoring.PrivacyNicknames:
			e.DisplayName = ""
		default:
			// names: keep nickname (legal names deferred to IQ.9 enrollment join;
			// until then nickname is the public label).
		}
		out[i] = e
	}
	return out
}

// CountActivePlayers returns non-removed players in a session.
func CountActivePlayers(ctx context.Context, pool *pgxpool.Pool, sessionID string) (int, error) {
	sid, err := uuid.Parse(sessionID)
	if err != nil {
		return 0, ErrSessionNotFound
	}
	var n int
	err = pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM quizgame.session_players
		WHERE session_id = $1 AND removed_at IS NULL`, sid).Scan(&n)
	return n, err
}

// LeaderboardRows is a thin wrapper kept for call sites that want Player structs.
func LeaderboardRows(ctx context.Context, pool *pgxpool.Pool, sessionID string, limit int) ([]Player, error) {
	entries, err := ComputeLeaderboard(ctx, pool, sessionID, limit)
	if err != nil {
		return nil, err
	}
	out := make([]Player, 0, len(entries))
	for _, e := range entries {
		out = append(out, Player{
			ID:         e.PlayerID,
			Nickname:   e.Nickname,
			TotalScore: e.TotalScore,
			Streak:      e.Streak,
		})
	}
	return out, nil
}
