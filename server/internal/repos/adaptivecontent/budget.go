package adaptivecontent

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// FeatureAdaptiveContent is the analytics.ai_usage_log feature key for ACE.
const FeatureAdaptiveContent = "adaptive_content"

// PeriodStartUTC returns the first day of the month (UTC) containing t.
func PeriodStartUTC(t time.Time) time.Time {
	u := t.UTC()
	return time.Date(u.Year(), u.Month(), 1, 0, 0, 0, 0, time.UTC)
}

// SumAdaptiveContentTokens returns total_tokens from ai_usage_log for the course
// and feature since periodStart (inclusive).
func SumAdaptiveContentTokens(ctx context.Context, pool *pgxpool.Pool, courseID uuid.UUID, periodStart time.Time) (int64, error) {
	if pool == nil {
		return 0, errors.New("adaptivecontent budget: nil pool")
	}
	var n int64
	err := pool.QueryRow(ctx, `
SELECT COALESCE(SUM(total_tokens), 0)::bigint
FROM analytics.ai_usage_log
WHERE course_id = $1
  AND feature = $2
  AND created_at >= $3
`, courseID, FeatureAdaptiveContent, periodStart).Scan(&n)
	return n, err
}

// SetTokensUsedPeriod writes the period cache on adaptive_content_settings.
// Creates a default settings row if missing.
func SetTokensUsedPeriod(ctx context.Context, pool *pgxpool.Pool, courseID uuid.UUID, periodStart time.Time, tokens int64) error {
	if tokens < 0 {
		tokens = 0
	}
	// period_start is DATE.
	day := time.Date(periodStart.Year(), periodStart.Month(), periodStart.Day(), 0, 0, 0, 0, time.UTC)
	_, err := pool.Exec(ctx, `
INSERT INTO course.adaptive_content_settings (course_id, budget_period_start, tokens_used_period, updated_at)
VALUES ($1, $2::date, $3, NOW())
ON CONFLICT (course_id) DO UPDATE SET
  budget_period_start = EXCLUDED.budget_period_start,
  tokens_used_period = EXCLUDED.tokens_used_period,
  updated_at = NOW()
`, courseID, day, tokens)
	return err
}

// AddTokensUsedPeriod increments the cache when a generation call completes.
// Resets the period if budget_period_start is null or in a prior month.
func AddTokensUsedPeriod(ctx context.Context, pool *pgxpool.Pool, courseID uuid.UUID, tokens int64, now time.Time) error {
	if tokens <= 0 {
		return nil
	}
	period := PeriodStartUTC(now)
	day := time.Date(period.Year(), period.Month(), period.Day(), 0, 0, 0, 0, time.UTC)
	_, err := pool.Exec(ctx, `
INSERT INTO course.adaptive_content_settings (course_id, budget_period_start, tokens_used_period, updated_at)
VALUES ($1, $2::date, $3, NOW())
ON CONFLICT (course_id) DO UPDATE SET
  tokens_used_period = CASE
    WHEN course.adaptive_content_settings.budget_period_start IS DISTINCT FROM EXCLUDED.budget_period_start
      THEN EXCLUDED.tokens_used_period
    ELSE course.adaptive_content_settings.tokens_used_period + EXCLUDED.tokens_used_period
  END,
  budget_period_start = EXCLUDED.budget_period_start,
  updated_at = NOW()
`, courseID, day, tokens)
	return err
}

// ReconcileTokensUsedPeriod sets the cache from ai_usage_log (source of truth).
func ReconcileTokensUsedPeriod(ctx context.Context, pool *pgxpool.Pool, courseID uuid.UUID, now time.Time) (int64, error) {
	period := PeriodStartUTC(now)
	n, err := SumAdaptiveContentTokens(ctx, pool, courseID, period)
	if err != nil {
		return 0, err
	}
	if err := SetTokensUsedPeriod(ctx, pool, courseID, period, n); err != nil {
		return n, err
	}
	return n, nil
}

// GetPlatformGenerationPaused reads settings.platform_app_settings.adaptive_content_generation_paused.
// Missing row ⇒ false (generation allowed).
func GetPlatformGenerationPaused(ctx context.Context, pool *pgxpool.Pool) (bool, error) {
	if pool == nil {
		return false, nil
	}
	var paused bool
	err := pool.QueryRow(ctx, `
SELECT COALESCE(adaptive_content_generation_paused, FALSE)
FROM settings.platform_app_settings
WHERE id = 1
`).Scan(&paused)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return paused, nil
}

// SetPlatformGenerationPaused sets the platform-wide generation pause switch.
func SetPlatformGenerationPaused(ctx context.Context, pool *pgxpool.Pool, paused bool) error {
	if pool == nil {
		return errors.New("adaptivecontent: nil pool")
	}
	if _, err := pool.Exec(ctx, `
INSERT INTO settings.platform_app_settings (id)
VALUES (1)
ON CONFLICT (id) DO NOTHING
`); err != nil {
		return err
	}
	_, err := pool.Exec(ctx, `
UPDATE settings.platform_app_settings
SET adaptive_content_generation_paused = $1, updated_at = NOW()
WHERE id = 1
`, paused)
	return err
}
