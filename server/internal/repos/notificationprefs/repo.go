package notificationprefs

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/lextures/lextures/server/internal/notificationevents"
	"github.com/lextures/lextures/server/internal/repos/user"
)

// Row is one notification preference.
type Row struct {
	EventType    string    `json:"eventType"`
	EmailEnabled bool      `json:"emailEnabled"`
	PushEnabled  bool      `json:"pushEnabled"`
	SmsEnabled   bool      `json:"smsEnabled"`
	DigestMode   string    `json:"digestMode"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

// ListForUser returns stored preferences, seeding defaults for missing event types.
func ListForUser(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID) ([]Row, error) {
	if err := EnsureDefaults(ctx, pool, userID); err != nil {
		return nil, err
	}
	rows, err := pool.Query(ctx, `
SELECT event_type, email_enabled, push_enabled, sms_enabled, digest_mode, updated_at
FROM settings.notification_preferences
WHERE user_id = $1
ORDER BY event_type
`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Row
	for rows.Next() {
		var r Row
		if err := rows.Scan(&r.EventType, &r.EmailEnabled, &r.PushEnabled, &r.SmsEnabled, &r.DigestMode, &r.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// EnsureDefaults inserts default rows for any missing event types.
func EnsureDefaults(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID) error {
	hasManaged, err := hasActiveManagedChild(ctx, pool, userID)
	if err != nil {
		return err
	}
	for _, et := range notificationevents.All {
		if et == notificationevents.ParentWeeklyProgress {
			if err := ensureWeeklyProgressDefault(ctx, pool, userID, hasManaged); err != nil {
				return err
			}
			continue
		}
		_, err := pool.Exec(ctx, `
INSERT INTO settings.notification_preferences (user_id, event_type)
VALUES ($1, $2)
ON CONFLICT (user_id, event_type) DO NOTHING
`, userID, et)
		if err != nil {
			return err
		}
	}
	return nil
}

// ensureWeeklyProgressDefault turns the weekly parent email on for homeschool
// accounts (a parent with an active managed learner) until the user saves or unsubscribes.
func ensureWeeklyProgressDefault(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID, hasManaged bool) error {
	_, err := pool.Exec(ctx, `
INSERT INTO settings.notification_preferences (user_id, event_type, email_enabled, push_enabled, sms_enabled, digest_mode)
VALUES ($1, $2, $3, false, false, 'instant')
ON CONFLICT (user_id, event_type) DO NOTHING
`, userID, notificationevents.ParentWeeklyProgress, hasManaged)
	if err != nil {
		return err
	}
	if !hasManaged {
		return nil
	}
	_, err = pool.Exec(ctx, `
UPDATE settings.notification_preferences
SET email_enabled = true
WHERE user_id = $1
  AND event_type = $2
  AND user_overridden = false
  AND email_enabled = false
`, userID, notificationevents.ParentWeeklyProgress)
	return err
}

func hasActiveManagedChild(ctx context.Context, pool *pgxpool.Pool, parentID uuid.UUID) (bool, error) {
	var exists bool
	err := pool.QueryRow(ctx, `
SELECT EXISTS (
  SELECT 1
  FROM "user".parent_student_links l
  JOIN "user".users s ON s.id = l.student_user_id
  WHERE l.parent_user_id = $1
    AND l.status = 'active'
    AND s.account_type = $2
    AND s.deactivated_at IS NULL
)
`, parentID, user.AccountTypeManaged).Scan(&exists)
	return exists, err
}

// Preference is the effective preference for one event type.
type Preference struct {
	EmailEnabled bool
	PushEnabled  bool
	SmsEnabled   bool
	DigestMode   string
}

// Get returns preference for one event (defaults if missing).
func Get(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID, eventType string) (Preference, error) {
	if err := EnsureDefaults(ctx, pool, userID); err != nil {
		return Preference{}, err
	}
	var p Preference
	err := pool.QueryRow(ctx, `
SELECT email_enabled, push_enabled, sms_enabled, digest_mode
FROM settings.notification_preferences
WHERE user_id = $1 AND event_type = $2
`, userID, eventType).Scan(&p.EmailEnabled, &p.PushEnabled, &p.SmsEnabled, &p.DigestMode)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Preference{EmailEnabled: true, PushEnabled: true, SmsEnabled: false, DigestMode: "instant"}, nil
		}
		return Preference{}, err
	}
	return p, nil
}

// UpsertItem updates one preference row.
func UpsertItem(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID, eventType string, emailEnabled, pushEnabled, smsEnabled *bool, digestMode *string) error {
	if err := EnsureDefaults(ctx, pool, userID); err != nil {
		return err
	}
	_, err := pool.Exec(ctx, `
INSERT INTO settings.notification_preferences (user_id, event_type, email_enabled, push_enabled, sms_enabled, digest_mode, user_overridden, updated_at)
VALUES ($1, $2,
  COALESCE($3, true),
  COALESCE($4, true),
  COALESCE($5, false),
  COALESCE($6, 'instant'),
  true,
  now())
ON CONFLICT (user_id, event_type) DO UPDATE SET
  email_enabled = COALESCE($3, settings.notification_preferences.email_enabled),
  push_enabled = COALESCE($4, settings.notification_preferences.push_enabled),
  sms_enabled = COALESCE($5, settings.notification_preferences.sms_enabled),
  digest_mode = COALESCE($6, settings.notification_preferences.digest_mode),
  user_overridden = true,
  updated_at = now()
`, userID, eventType, emailEnabled, pushEnabled, smsEnabled, digestMode)
	return err
}

// SetEmailEnabled sets email_enabled for one event (unsubscribe).
func SetEmailEnabled(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID, eventType string, enabled bool) error {
	return UpsertItem(ctx, pool, userID, eventType, &enabled, nil, nil, nil)
}
