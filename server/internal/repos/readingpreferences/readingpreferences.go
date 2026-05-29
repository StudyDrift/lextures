// Package readingpreferences stores per-user TTS / reading preferences (plan 12.8).
package readingpreferences

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Row is a user's reading preferences.
type Row struct {
	UserID       uuid.UUID
	TTSEnabled   bool
	TTSSpeed     float64
	TTSVoiceName *string
	UpdatedAt    time.Time
}

// Defaults returns platform defaults for a new user.
func Defaults() Row {
	return Row{
		TTSEnabled: false,
		TTSSpeed:   1.0,
	}
}

// Get returns preferences for userID, or defaults when no row exists.
func Get(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID) (Row, error) {
	const q = `
SELECT user_id, tts_enabled, (tts_speed)::double precision, tts_voice_name, updated_at
FROM "user".user_reading_preferences
WHERE user_id = $1`
	var r Row
	var voice *string
	err := pool.QueryRow(ctx, q, userID).Scan(&r.UserID, &r.TTSEnabled, &r.TTSSpeed, &voice, &r.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		d := Defaults()
		d.UserID = userID
		return d, nil
	}
	if err != nil {
		return Row{}, err
	}
	r.TTSVoiceName = voice
	return r, nil
}

// Patch updates TTS fields; nil pointers leave the column unchanged.
func Patch(
	ctx context.Context,
	pool *pgxpool.Pool,
	userID uuid.UUID,
	ttsEnabled *bool,
	ttsSpeed *float64,
	ttsVoiceName **string,
) (Row, error) {
	cur, err := Get(ctx, pool, userID)
	if err != nil {
		return Row{}, err
	}
	next := cur
	if ttsEnabled != nil {
		next.TTSEnabled = *ttsEnabled
	}
	if ttsSpeed != nil {
		next.TTSSpeed = *ttsSpeed
	}
	if ttsVoiceName != nil {
		next.TTSVoiceName = *ttsVoiceName
	}
	const q = `
INSERT INTO "user".user_reading_preferences (user_id, tts_enabled, tts_speed, tts_voice_name, updated_at)
VALUES ($1, $2, $3::numeric, $4, NOW())
ON CONFLICT (user_id) DO UPDATE SET
  tts_enabled = EXCLUDED.tts_enabled,
  tts_speed = EXCLUDED.tts_speed,
  tts_voice_name = EXCLUDED.tts_voice_name,
  updated_at = NOW()
RETURNING user_id, tts_enabled, (tts_speed)::double precision, tts_voice_name, updated_at`
	var voice *string
	var r Row
	err = pool.QueryRow(ctx, q, userID, next.TTSEnabled, next.TTSSpeed, next.TTSVoiceName).Scan(
		&r.UserID, &r.TTSEnabled, &r.TTSSpeed, &voice, &r.UpdatedAt,
	)
	if err != nil {
		return Row{}, err
	}
	r.TTSVoiceName = voice
	return r, nil
}
