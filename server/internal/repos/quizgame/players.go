package quizgame

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Player is one session_players row.
type Player struct {
	ID         string
	SessionID  string
	UserID     *string
	Nickname   string
	TotalScore int
	Streak     int
	Connected  bool
	JoinedAt   time.Time
	RemovedAt  *time.Time
	LastSeenAt *time.Time
}

// AddPlayerInput joins a player to a lobby/running session.
type AddPlayerInput struct {
	SessionID  string
	UserID     *uuid.UUID
	Nickname   string
	ClientMeta json.RawMessage
}

// AddPlayerResult includes the raw reconnect token (shown once).
type AddPlayerResult struct {
	Player      Player
	PlayerToken string // raw; store hashed in DB
	Rejoined    bool
}

// AddPlayer creates a player with a reconnect token. Enrolled-only for v1 (caller enforces).
// If the same enrolled user already has a non-removed row, rotates their token and returns
// that player (rejoin) instead of inserting a duplicate.
func AddPlayer(ctx context.Context, pool *pgxpool.Pool, in AddPlayerInput) (*AddPlayerResult, error) {
	nick, err := ValidateNickname(in.Nickname)
	if err != nil {
		return nil, ErrNicknameInvalid
	}
	sess, err := GetSession(ctx, pool, in.SessionID)
	if err != nil {
		return nil, err
	}
	if sess.Status == "ended" || sess.Status == "abandoned" {
		return nil, ErrSessionEnded
	}

	if in.UserID != nil {
		existing, err := GetPlayerByUser(ctx, pool, in.SessionID, *in.UserID)
		if err == nil && existing != nil {
			raw, err := RotatePlayerToken(ctx, pool, existing.ID)
			if err != nil {
				return nil, err
			}
			_ = TouchPlayerSeen(ctx, pool, existing.ID, in.ClientMeta)
			p, err := GetPlayer(ctx, pool, existing.ID)
			if err != nil {
				return nil, err
			}
			return &AddPlayerResult{Player: *p, PlayerToken: raw, Rejoined: true}, nil
		}
		if err != nil && !errors.Is(err, ErrPlayerNotFound) {
			return nil, err
		}
	}

	raw, err := NewPlayerToken()
	if err != nil {
		return nil, err
	}
	hashed := HashPlayerToken(raw)
	sid, err := uuid.Parse(in.SessionID)
	if err != nil {
		return nil, ErrSessionNotFound
	}
	var user any
	if in.UserID != nil {
		user = *in.UserID
	}
	meta := in.ClientMeta
	if len(meta) == 0 {
		meta = json.RawMessage(`{}`)
	}
	var id uuid.UUID
	err = pool.QueryRow(ctx, `
		INSERT INTO quizgame.session_players
			(session_id, user_id, nickname, player_token, connected, last_seen_at, client_meta)
		VALUES ($1, $2, $3, $4, TRUE, NOW(), $5::jsonb)
		RETURNING id`, sid, user, nick, hashed, []byte(meta)).Scan(&id)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, ErrPlayerExists
		}
		return nil, err
	}
	p, err := GetPlayer(ctx, pool, id.String())
	if err != nil {
		return nil, err
	}
	return &AddPlayerResult{Player: *p, PlayerToken: raw}, nil
}

// GetPlayerByUser finds a non-removed player for an enrolled user in a session.
func GetPlayerByUser(ctx context.Context, pool *pgxpool.Pool, sessionID string, userID uuid.UUID) (*Player, error) {
	sid, err := uuid.Parse(sessionID)
	if err != nil {
		return nil, ErrPlayerNotFound
	}
	row := pool.QueryRow(ctx, `
		SELECT id, session_id, user_id, nickname, total_score, streak, connected, joined_at, removed_at, last_seen_at
		FROM quizgame.session_players
		WHERE session_id = $1 AND user_id = $2 AND removed_at IS NULL
		ORDER BY joined_at ASC
		LIMIT 1`, sid, userID)
	return scanPlayer(row)
}

// RotatePlayerToken issues a new reconnect secret for an existing player.
func RotatePlayerToken(ctx context.Context, pool *pgxpool.Pool, playerID string) (string, error) {
	id, err := uuid.Parse(playerID)
	if err != nil {
		return "", ErrPlayerNotFound
	}
	raw, err := NewPlayerToken()
	if err != nil {
		return "", err
	}
	hashed := HashPlayerToken(raw)
	tag, err := pool.Exec(ctx, `
		UPDATE quizgame.session_players
		SET player_token = $2, connected = TRUE, last_seen_at = NOW()
		WHERE id = $1 AND removed_at IS NULL`, id, hashed)
	if err != nil {
		return "", err
	}
	if tag.RowsAffected() == 0 {
		return "", ErrPlayerNotFound
	}
	return raw, nil
}

// TouchPlayerSeen updates last_seen_at and optional coarse client_meta.
func TouchPlayerSeen(ctx context.Context, pool *pgxpool.Pool, playerID string, clientMeta json.RawMessage) error {
	id, err := uuid.Parse(playerID)
	if err != nil {
		return ErrPlayerNotFound
	}
	if len(clientMeta) == 0 {
		_, err = pool.Exec(ctx, `UPDATE quizgame.session_players SET last_seen_at = NOW() WHERE id = $1`, id)
		return err
	}
	_, err = pool.Exec(ctx, `
		UPDATE quizgame.session_players
		SET last_seen_at = NOW(), client_meta = $2::jsonb
		WHERE id = $1`, id, []byte(clientMeta))
	return err
}

// GetPlayer loads a player by id.
func GetPlayer(ctx context.Context, pool *pgxpool.Pool, playerID string) (*Player, error) {
	id, err := uuid.Parse(playerID)
	if err != nil {
		return nil, ErrPlayerNotFound
	}
	row := pool.QueryRow(ctx, `
		SELECT id, session_id, user_id, nickname, total_score, streak, connected, joined_at, removed_at, last_seen_at
		FROM quizgame.session_players WHERE id = $1`, id)
	return scanPlayer(row)
}

// GetPlayerByToken finds a player by raw reconnect token.
func GetPlayerByToken(ctx context.Context, pool *pgxpool.Pool, sessionID, rawToken string) (*Player, error) {
	sid, err := uuid.Parse(sessionID)
	if err != nil {
		return nil, ErrPlayerNotFound
	}
	hashed := HashPlayerToken(rawToken)
	row := pool.QueryRow(ctx, `
		SELECT id, session_id, user_id, nickname, total_score, streak, connected, joined_at, removed_at, last_seen_at
		FROM quizgame.session_players
		WHERE session_id = $1 AND player_token = $2 AND removed_at IS NULL`, sid, hashed)
	return scanPlayer(row)
}

func scanPlayer(row pgx.Row) (*Player, error) {
	var p Player
	var id, sid uuid.UUID
	var uid uuid.NullUUID
	var lastSeen *time.Time
	err := row.Scan(&id, &sid, &uid, &p.Nickname, &p.TotalScore, &p.Streak, &p.Connected, &p.JoinedAt, &p.RemovedAt, &lastSeen)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrPlayerNotFound
	}
	if err != nil {
		return nil, err
	}
	p.ID = id.String()
	p.SessionID = sid.String()
	p.LastSeenAt = lastSeen
	if uid.Valid {
		u := uid.UUID.String()
		p.UserID = &u
	}
	return &p, nil
}

// ListPlayers returns non-removed players for a session.
func ListPlayers(ctx context.Context, pool *pgxpool.Pool, sessionID string) ([]Player, error) {
	sid, err := uuid.Parse(sessionID)
	if err != nil {
		return nil, err
	}
	rows, err := pool.Query(ctx, `
		SELECT id, session_id, user_id, nickname, total_score, streak, connected, joined_at, removed_at, last_seen_at
		FROM quizgame.session_players
		WHERE session_id = $1 AND removed_at IS NULL
		ORDER BY joined_at ASC`, sid)
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

// SetPlayerConnected updates the connected flag.
func SetPlayerConnected(ctx context.Context, pool *pgxpool.Pool, playerID string, connected bool) error {
	id, err := uuid.Parse(playerID)
	if err != nil {
		return err
	}
	_, err = pool.Exec(ctx, `
		UPDATE quizgame.session_players
		SET connected = $2, last_seen_at = CASE WHEN $2 THEN NOW() ELSE last_seen_at END
		WHERE id = $1`, id, connected)
	return err
}

// KickPlayer soft-removes a player (IQ.9 hook).
func KickPlayer(ctx context.Context, pool *pgxpool.Pool, sessionID, playerID string) error {
	sid, err := uuid.Parse(sessionID)
	if err != nil {
		return err
	}
	pid, err := uuid.Parse(playerID)
	if err != nil {
		return ErrPlayerNotFound
	}
	tag, err := pool.Exec(ctx, `
		UPDATE quizgame.session_players SET removed_at = NOW(), connected = FALSE
		WHERE id = $1 AND session_id = $2 AND removed_at IS NULL`, pid, sid)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrPlayerNotFound
	}
	return nil
}

// AddPlayerScore increments total_score and optionally streak.
func AddPlayerScore(ctx context.Context, tx pgx.Tx, playerID string, points int, correct bool) error {
	id, err := uuid.Parse(playerID)
	if err != nil {
		return err
	}
	if correct {
		_, err = tx.Exec(ctx, `
			UPDATE quizgame.session_players
			SET total_score = total_score + $2, streak = streak + 1
			WHERE id = $1`, id, points)
	} else {
		_, err = tx.Exec(ctx, `
			UPDATE quizgame.session_players
			SET total_score = total_score + $2, streak = 0
			WHERE id = $1`, id, points)
	}
	return err
}
