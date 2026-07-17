package quizgame

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lextures/lextures/server/internal/quizgame/scoring"
)

var (
	ErrPowerUpsDisabled = errors.New("quizgame: power-ups disabled")
	ErrPowerUpInvalid   = errors.New("quizgame: invalid power-up")
)

// ValidatePowerUpClaim checks server-side eligibility (FR-8). Does not consume.
func ValidatePowerUpClaim(ctx context.Context, pool *pgxpool.Pool, sess *Session, player *Player, questionIndex int, kind string) (bool, error) {
	cfg := scoring.ResolveConfig(sess.ScoringProfile, scoring.ParseConfigJSON(sess.ScoringConfig))
	if !cfg.PowerUpsEnabled {
		return false, nil
	}
	switch kind {
	case scoring.PowerUpDoubleOrNothing:
		used, err := PlayerUsedPowerUpOnQuestion(ctx, pool, sess.ID, player.ID, questionIndex, kind)
		if err != nil {
			return false, err
		}
		return !used, nil
	case scoring.PowerUpShield:
		used, err := PlayerUsedPowerUpKind(ctx, pool, sess.ID, player.ID, scoring.PowerUpShield)
		if err != nil {
			return false, err
		}
		return !used, nil
	default:
		return false, ErrPowerUpInvalid
	}
}

// PlayerUsedPowerUpKind reports whether the player already used this kind in the session.
func PlayerUsedPowerUpKind(ctx context.Context, pool *pgxpool.Pool, sessionID, playerID, kind string) (bool, error) {
	sid, err := uuid.Parse(sessionID)
	if err != nil {
		return false, ErrSessionNotFound
	}
	pid, err := uuid.Parse(playerID)
	if err != nil {
		return false, ErrPlayerNotFound
	}
	var n int
	err = pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM quizgame.player_powerups
		WHERE session_id = $1 AND player_id = $2 AND kind = $3`, sid, pid, kind).Scan(&n)
	return n > 0, err
}

// PlayerUsedPowerUpOnQuestion reports a claim for this question+kind.
func PlayerUsedPowerUpOnQuestion(ctx context.Context, pool *pgxpool.Pool, sessionID, playerID string, questionIndex int, kind string) (bool, error) {
	sid, err := uuid.Parse(sessionID)
	if err != nil {
		return false, ErrSessionNotFound
	}
	pid, err := uuid.Parse(playerID)
	if err != nil {
		return false, ErrPlayerNotFound
	}
	var n int
	err = pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM quizgame.player_powerups
		WHERE session_id = $1 AND player_id = $2 AND question_index = $3 AND kind = $4`,
		sid, pid, questionIndex, kind).Scan(&n)
	return n > 0, err
}

// RecordPowerUp inserts into the ledger (idempotent on PK).
func RecordPowerUp(ctx context.Context, tx pgx.Tx, sessionID, playerID string, questionIndex int, kind string) error {
	sid, err := uuid.Parse(sessionID)
	if err != nil {
		return err
	}
	pid, err := uuid.Parse(playerID)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO quizgame.player_powerups (session_id, player_id, question_index, kind)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT DO NOTHING`, sid, pid, questionIndex, kind)
	return err
}

// ClaimPowerUp validates a pre-answer opt-in (WS powerup frame). Persistence happens at answer time.
func ClaimPowerUp(ctx context.Context, pool *pgxpool.Pool, sessionID, playerID string, questionIndex int, kind string) error {
	sess, err := GetSession(ctx, pool, sessionID)
	if err != nil {
		return err
	}
	cfg := scoring.ResolveConfig(sess.ScoringProfile, scoring.ParseConfigJSON(sess.ScoringConfig))
	if !cfg.PowerUpsEnabled {
		return ErrPowerUpsDisabled
	}
	player, err := GetPlayer(ctx, pool, playerID)
	if err != nil || player == nil {
		return ErrPlayerNotFound
	}
	ok, err := ValidatePowerUpClaim(ctx, pool, sess, player, questionIndex, kind)
	if err != nil {
		return err
	}
	if !ok {
		return ErrPowerUpInvalid
	}
	return nil
}
