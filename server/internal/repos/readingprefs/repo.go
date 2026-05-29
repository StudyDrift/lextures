// Package readingprefs stores per-user reading and accessibility preferences (plan 12.6+).
package readingprefs

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Row is the STT-related slice of user_reading_preferences (extended by later plans).
type Row struct {
	UserID      uuid.UUID
	STTEnabled  bool
	STTLanguage string
	UpdatedAt   time.Time
}

const defaultSTTLanguage = "en-US"

// Get returns preferences for a user, creating defaults when missing.
func Get(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID) (Row, error) {
	const q = `
SELECT user_id, stt_enabled, stt_language, updated_at
FROM settings.user_reading_preferences
WHERE user_id = $1`
	var r Row
	err := pool.QueryRow(ctx, q, userID).Scan(&r.UserID, &r.STTEnabled, &r.STTLanguage, &r.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Row{
			UserID:      userID,
			STTEnabled:  false,
			STTLanguage: defaultSTTLanguage,
			UpdatedAt:   time.Now().UTC(),
		}, nil
	}
	if err != nil {
		return Row{}, err
	}
	return r, nil
}

// Patch updates one or more fields; upserts the row when absent.
func Patch(
	ctx context.Context,
	pool *pgxpool.Pool,
	userID uuid.UUID,
	sttEnabled *bool,
	sttLanguage *string,
) (Row, error) {
	cur, err := Get(ctx, pool, userID)
	if err != nil {
		return Row{}, err
	}
	enabled := cur.STTEnabled
	lang := cur.STTLanguage
	if sttEnabled != nil {
		enabled = *sttEnabled
	}
	if sttLanguage != nil {
		trimmed := *sttLanguage
		if trimmed == "" {
			trimmed = defaultSTTLanguage
		}
		lang = trimmed
	}
	const q = `
INSERT INTO settings.user_reading_preferences (user_id, stt_enabled, stt_language, updated_at)
VALUES ($1, $2, $3, NOW())
ON CONFLICT (user_id) DO UPDATE
SET stt_enabled = EXCLUDED.stt_enabled,
    stt_language = EXCLUDED.stt_language,
    updated_at = NOW()
RETURNING user_id, stt_enabled, stt_language, updated_at`
	var r Row
	err = pool.QueryRow(ctx, q, userID, enabled, lang).Scan(&r.UserID, &r.STTEnabled, &r.STTLanguage, &r.UpdatedAt)
	return r, err
}
