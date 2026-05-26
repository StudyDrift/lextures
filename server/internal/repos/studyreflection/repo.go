// Package studyreflection persists study goals, journal entries, and coaching tips (plan 9.9).
package studyreflection

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Goal is a user's weekly study goal and opt-in preference.
type Goal struct {
	UserID      uuid.UUID
	WeeklyHours float32
	OptedIn     bool
	UpdatedAt   time.Time
}

// JournalEntry is one private reflection note.
type JournalEntry struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	CourseID  *uuid.UUID
	EntryText string
	CreatedAt time.Time
}

// CoachingTip is a weekly AI or fallback coaching message.
type CoachingTip struct {
	ID          uuid.UUID
	UserID      uuid.UUID
	TipText     string
	WeekOf      time.Time
	DeliveredAt *time.Time
	Rating      *int16
}

// GetGoal returns the user's goal row or nil if unset.
func GetGoal(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID) (*Goal, error) {
	var g Goal
	err := pool.QueryRow(ctx, `
SELECT user_id, weekly_hours, opted_in, updated_at
FROM analytics.study_goals
WHERE user_id = $1
`, userID).Scan(&g.UserID, &g.WeeklyHours, &g.OptedIn, &g.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &g, nil
}

// UpsertGoal saves weekly hours and opt-in status.
func UpsertGoal(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID, weeklyHours float32, optedIn bool) error {
	_, err := pool.Exec(ctx, `
INSERT INTO analytics.study_goals (user_id, weekly_hours, opted_in, updated_at)
VALUES ($1, $2, $3, now())
ON CONFLICT (user_id) DO UPDATE SET
    weekly_hours = EXCLUDED.weekly_hours,
    opted_in = EXCLUDED.opted_in,
    updated_at = now()
`, userID, weeklyHours, optedIn)
	return err
}

// ListJournal returns paginated journal entries for a user.
func ListJournal(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID, limit, offset int) ([]JournalEntry, error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 100 {
		limit = 100
	}
	rows, err := pool.Query(ctx, `
SELECT id, user_id, course_id, entry_text, created_at
FROM analytics.reflection_journal
WHERE user_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3
`, userID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []JournalEntry
	for rows.Next() {
		var e JournalEntry
		if err := rows.Scan(&e.ID, &e.UserID, &e.CourseID, &e.EntryText, &e.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// InsertJournal creates a journal entry.
func InsertJournal(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID, courseID *uuid.UUID, text string) (uuid.UUID, error) {
	var id uuid.UUID
	err := pool.QueryRow(ctx, `
INSERT INTO analytics.reflection_journal (user_id, course_id, entry_text)
VALUES ($1, $2, $3)
RETURNING id
`, userID, courseID, text).Scan(&id)
	return id, err
}

// DeleteJournal removes an entry owned by the user.
func DeleteJournal(ctx context.Context, pool *pgxpool.Pool, userID, entryID uuid.UUID) (bool, error) {
	tag, err := pool.Exec(ctx, `
DELETE FROM analytics.reflection_journal WHERE id = $1 AND user_id = $2
`, entryID, userID)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}

// LatestCoachingTip returns the most recent tip for the user.
func LatestCoachingTip(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID) (*CoachingTip, error) {
	var tip CoachingTip
	err := pool.QueryRow(ctx, `
SELECT id, user_id, tip_text, week_of, delivered_at, rating
FROM analytics.coaching_tips
WHERE user_id = $1
ORDER BY week_of DESC
LIMIT 1
`, userID).Scan(&tip.ID, &tip.UserID, &tip.TipText, &tip.WeekOf, &tip.DeliveredAt, &tip.Rating)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &tip, nil
}

// ListCoachingTips returns tip history for a user.
func ListCoachingTips(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID, limit int) ([]CoachingTip, error) {
	if limit <= 0 {
		limit = 12
	}
	rows, err := pool.Query(ctx, `
SELECT id, user_id, tip_text, week_of, delivered_at, rating
FROM analytics.coaching_tips
WHERE user_id = $1
ORDER BY week_of DESC
LIMIT $2
`, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []CoachingTip
	for rows.Next() {
		var tip CoachingTip
		if err := rows.Scan(&tip.ID, &tip.UserID, &tip.TipText, &tip.WeekOf, &tip.DeliveredAt, &tip.Rating); err != nil {
			return nil, err
		}
		out = append(out, tip)
	}
	return out, rows.Err()
}

// UpsertCoachingTip stores a weekly tip (idempotent per user/week).
func UpsertCoachingTip(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID, weekOf time.Time, tipText string) error {
	_, err := pool.Exec(ctx, `
INSERT INTO analytics.coaching_tips (user_id, tip_text, week_of, delivered_at)
VALUES ($1, $2, $3::date, now())
ON CONFLICT (user_id, week_of) DO UPDATE SET
    tip_text = EXCLUDED.tip_text,
    delivered_at = COALESCE(analytics.coaching_tips.delivered_at, now())
`, userID, tipText, weekOf)
	return err
}

// RateCoachingTip stores thumbs up/down for a tip.
func RateCoachingTip(ctx context.Context, pool *pgxpool.Pool, userID, tipID uuid.UUID, rating int16) (bool, error) {
	tag, err := pool.Exec(ctx, `
UPDATE analytics.coaching_tips SET rating = $3
WHERE id = $1 AND user_id = $2
`, tipID, userID, rating)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}

// ListOptedInUserIDs returns enrolled students who opted in to coaching for batch generation.
func ListOptedInUserIDs(ctx context.Context, pool *pgxpool.Pool, limit int) ([]uuid.UUID, error) {
	if limit <= 0 {
		limit = 500
	}
	rows, err := pool.Query(ctx, `
SELECT DISTINCT g.user_id
FROM analytics.study_goals g
JOIN course.course_enrollments e ON e.user_id = g.user_id AND e.active = true
WHERE g.opted_in = true
LIMIT $1
`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// HasCoachingTipForWeek reports whether a tip already exists for user/week.
func HasCoachingTipForWeek(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID, weekOf time.Time) (bool, error) {
	var n int
	err := pool.QueryRow(ctx, `
SELECT COUNT(*) FROM analytics.coaching_tips WHERE user_id = $1 AND week_of = $2::date
`, userID, weekOf).Scan(&n)
	return n > 0, err
}
