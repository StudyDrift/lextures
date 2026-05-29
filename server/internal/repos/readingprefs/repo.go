// Package readingprefs manages per-user accessibility reading preferences (plan 12.7).
package readingprefs

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Row is one user_reading_preferences row.
type Row struct {
	UserID       uuid.UUID `json:"userId"`
	HighContrast bool      `json:"highContrast"`
	ReduceMotion bool      `json:"reduceMotion"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

// Get returns the preferences row for userID, or a zero-value Row (all false) if none exists.
func Get(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID) (*Row, error) {
	r := &Row{UserID: userID}
	err := pool.QueryRow(ctx, `
SELECT high_contrast, reduce_motion, updated_at
FROM settings.user_reading_preferences
WHERE user_id = $1
`, userID).Scan(&r.HighContrast, &r.ReduceMotion, &r.UpdatedAt)
	if err == pgx.ErrNoRows {
		return r, nil
	}
	if err != nil {
		return nil, err
	}
	return r, nil
}

// Upsert inserts or updates the preferences for userID and returns the saved row.
func Upsert(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID, highContrast, reduceMotion bool) (*Row, error) {
	r := &Row{UserID: userID}
	err := pool.QueryRow(ctx, `
INSERT INTO settings.user_reading_preferences (user_id, high_contrast, reduce_motion, updated_at)
VALUES ($1, $2, $3, NOW())
ON CONFLICT (user_id) DO UPDATE
    SET high_contrast = EXCLUDED.high_contrast,
        reduce_motion = EXCLUDED.reduce_motion,
        updated_at    = NOW()
RETURNING high_contrast, reduce_motion, updated_at
`, userID, highContrast, reduceMotion).Scan(&r.HighContrast, &r.ReduceMotion, &r.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return r, nil
}
