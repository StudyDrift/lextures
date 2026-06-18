// Package studyreminders persists learner reminder preferences and send logs (plan 15.10).
package studyreminders

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Config is a user's study reminder preferences.
type Config struct {
	UserID           uuid.UUID
	DailyGoalMinutes int
	ReminderTime     time.Time // time-of-day component used
	ReminderChannels []string
	PausedUntil      *time.Time
	WeeklySummary    bool
	Enabled          bool
	UpdatedAt        time.Time
}

// Candidate is an enabled reminder config with the user's timezone.
type Candidate struct {
	UserID           uuid.UUID
	DailyGoalMinutes int
	ReminderTime     time.Time
	ReminderChannels []string
	WeeklySummary    bool
	PausedUntil      *time.Time
	Timezone         *string
}

const (
	ReminderDaily         = "daily"
	ReminderStreakAtRisk  = "streak_at_risk"
	ReminderWeeklySummary = "weekly_summary"
)

// Get returns the user's config or nil when unset.
func Get(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID) (*Config, error) {
	var c Config
	var paused sql.NullTime
	err := pool.QueryRow(ctx, `
SELECT user_id, daily_goal_minutes, reminder_time, reminder_channels, paused_until,
       weekly_summary, enabled, updated_at
FROM studyreminders.configs
WHERE user_id = $1
`, userID).Scan(
		&c.UserID, &c.DailyGoalMinutes, &c.ReminderTime, &c.ReminderChannels, &paused,
		&c.WeeklySummary, &c.Enabled, &c.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if paused.Valid {
		d := time.Date(paused.Time.Year(), paused.Time.Month(), paused.Time.Day(), 0, 0, 0, 0, time.UTC)
		c.PausedUntil = &d
	}
	return &c, nil
}

// Upsert saves reminder preferences.
func Upsert(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID, dailyGoal int, reminderTime time.Time, channels []string, weeklySummary, enabled bool) error {
	if len(channels) == 0 {
		channels = []string{"email"}
	}
	_, err := pool.Exec(ctx, `
INSERT INTO studyreminders.configs (
    user_id, daily_goal_minutes, reminder_time, reminder_channels, weekly_summary, enabled, updated_at
) VALUES ($1, $2, $3::time, $4, $5, $6, NOW())
ON CONFLICT (user_id) DO UPDATE SET
    daily_goal_minutes = EXCLUDED.daily_goal_minutes,
    reminder_time = EXCLUDED.reminder_time,
    reminder_channels = EXCLUDED.reminder_channels,
    weekly_summary = EXCLUDED.weekly_summary,
    enabled = EXCLUDED.enabled,
    updated_at = NOW()
`, userID, dailyGoal, reminderTime.Format("15:04:05"), channels, weeklySummary, enabled)
	return err
}

// SetEnabled toggles reminders on or off.
func SetEnabled(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID, enabled bool) error {
	_, err := pool.Exec(ctx, `
INSERT INTO studyreminders.configs (user_id, enabled, updated_at)
VALUES ($1, $2, NOW())
ON CONFLICT (user_id) DO UPDATE SET enabled = EXCLUDED.enabled, updated_at = NOW()
`, userID, enabled)
	return err
}

// PauseUntil sets paused_until for N days from today (user-local date passed in).
func PauseUntil(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID, until time.Time) error {
	_, err := pool.Exec(ctx, `
INSERT INTO studyreminders.configs (user_id, paused_until, updated_at)
VALUES ($1, $2::date, NOW())
ON CONFLICT (user_id) DO UPDATE SET paused_until = EXCLUDED.paused_until, updated_at = NOW()
`, userID, until.Format("2006-01-02"))
	return err
}

// ListEnabledCandidates returns users with reminders enabled.
func ListEnabledCandidates(ctx context.Context, pool *pgxpool.Pool, limit int) ([]Candidate, error) {
	if limit <= 0 {
		limit = 500
	}
	rows, err := pool.Query(ctx, `
SELECT c.user_id, c.daily_goal_minutes, c.reminder_time, c.reminder_channels, c.weekly_summary, c.paused_until, u.timezone
FROM studyreminders.configs c
JOIN "user".users u ON u.id = c.user_id
WHERE c.enabled = TRUE
ORDER BY c.user_id
LIMIT $1
`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Candidate
	for rows.Next() {
		var c Candidate
		var paused sql.NullTime
		if err := rows.Scan(&c.UserID, &c.DailyGoalMinutes, &c.ReminderTime, &c.ReminderChannels, &c.WeeklySummary, &paused, &c.Timezone); err != nil {
			return nil, err
		}
		if paused.Valid {
			d := time.Date(paused.Time.Year(), paused.Time.Month(), paused.Time.Day(), 0, 0, 0, 0, time.UTC)
			c.PausedUntil = &d
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// WasSentToday reports whether a reminder type was already delivered today on a channel.
func WasSentToday(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID, sendDate time.Time, reminderType, channel string) (bool, error) {
	var exists bool
	err := pool.QueryRow(ctx, `
SELECT EXISTS (
    SELECT 1 FROM studyreminders.send_log
    WHERE user_id = $1 AND send_date = $2::date AND reminder_type = $3 AND channel = $4
)
`, userID, sendDate.Format("2006-01-02"), reminderType, channel).Scan(&exists)
	return exists, err
}

// LogSend records a successful reminder delivery.
func LogSend(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID, sendDate time.Time, reminderType, channel, idempotencyKey string) error {
	_, err := pool.Exec(ctx, `
INSERT INTO studyreminders.send_log (user_id, send_date, reminder_type, channel, idempotency_key)
VALUES ($1, $2::date, $3, $4, $5)
ON CONFLICT (user_id, send_date, reminder_type, channel) DO NOTHING
`, userID, sendDate.Format("2006-01-02"), reminderType, channel, idempotencyKey)
	return err
}
