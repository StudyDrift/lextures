// Package familysignin persists managed-learner PIN / picture sign-ins and
// parent-registered family devices (issue #771).
package familysignin

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// SignIn is a managed learner's configured PIN or picture login.
type SignIn struct {
	UserID         uuid.UUID
	Method         string
	SecretHash     string
	FailedAttempts int
	LockedUntil    *time.Time
	UpdatedAt      time.Time
}

// Status is the non-secret summary used by parent and picker UIs.
type Status struct {
	Method      string
	LockedUntil *time.Time
}

// Device is a browser a parent marked as a shared family device.
type Device struct {
	ID           uuid.UUID
	ParentUserID uuid.UUID
	Label        string
	CreatedAt    time.Time
	LastUsedAt   *time.Time
}

// PickerLearner is one entry in the "Who's learning?" picker.
type PickerLearner struct {
	ID          uuid.UUID
	DisplayName string
	Method      string // "", "pin", or "picture"
	LockedUntil *time.Time
}

// UpsertSignIn sets (or replaces) the learner's login and clears any lockout.
func UpsertSignIn(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID, method, secretHash string, updatedBy uuid.UUID) error {
	_, err := pool.Exec(ctx, `
INSERT INTO "user".managed_learner_sign_ins (user_id, method, secret_hash, updated_by)
VALUES ($1, $2, $3, $4)
ON CONFLICT (user_id) DO UPDATE
SET method = EXCLUDED.method,
    secret_hash = EXCLUDED.secret_hash,
    updated_by = EXCLUDED.updated_by,
    failed_attempts = 0,
    locked_until = NULL,
    updated_at = now()
`, userID, method, secretHash, updatedBy)
	return err
}

// DeleteSignIn disables PIN / picture login for the learner.
func DeleteSignIn(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID) error {
	_, err := pool.Exec(ctx, `DELETE FROM "user".managed_learner_sign_ins WHERE user_id = $1`, userID)
	return err
}

// GetSignIn returns the learner's login, or nil when none is configured.
func GetSignIn(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID) (*SignIn, error) {
	var s SignIn
	err := pool.QueryRow(ctx, `
SELECT user_id, method, secret_hash, failed_attempts, locked_until, updated_at
FROM "user".managed_learner_sign_ins
WHERE user_id = $1
`, userID).Scan(&s.UserID, &s.Method, &s.SecretHash, &s.FailedAttempts, &s.LockedUntil, &s.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &s, nil
}

// StatusesFor returns sign-in summaries keyed by learner id (missing = not configured).
func StatusesFor(ctx context.Context, pool *pgxpool.Pool, userIDs []uuid.UUID) (map[uuid.UUID]Status, error) {
	out := make(map[uuid.UUID]Status, len(userIDs))
	if len(userIDs) == 0 {
		return out, nil
	}
	rows, err := pool.Query(ctx, `
SELECT user_id, method, locked_until
FROM "user".managed_learner_sign_ins
WHERE user_id = ANY($1)
`, userIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var id uuid.UUID
		var st Status
		if err := rows.Scan(&id, &st.Method, &st.LockedUntil); err != nil {
			return nil, err
		}
		out[id] = st
	}
	return out, rows.Err()
}

// RecordFailure increments the failed-attempt counter. When the counter reaches
// maxAttempts the learner is locked until now+lockFor and the counter resets.
// justLocked is true only for the attempt that triggered the lock.
func RecordFailure(
	ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID, maxAttempts int, lockFor time.Duration, now time.Time,
) (attempts int, lockedUntil *time.Time, justLocked bool, err error) {
	lockAt := now.UTC().Add(lockFor)
	err = pool.QueryRow(ctx, `
UPDATE "user".managed_learner_sign_ins
SET failed_attempts = CASE WHEN failed_attempts + 1 >= $2::int THEN 0 ELSE failed_attempts + 1 END,
    locked_until    = CASE WHEN failed_attempts + 1 >= $2::int THEN $3::timestamptz ELSE locked_until END
WHERE user_id = $1
RETURNING failed_attempts, locked_until, COALESCE(locked_until = $3::timestamptz, FALSE)
`, userID, maxAttempts, lockAt).Scan(&attempts, &lockedUntil, &justLocked)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, nil, false, nil
	}
	return attempts, lockedUntil, justLocked, err
}

// ResetFailures clears the failed-attempt counter and lockout after a successful sign-in.
func ResetFailures(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID) error {
	_, err := pool.Exec(ctx, `
UPDATE "user".managed_learner_sign_ins
SET failed_attempts = 0, locked_until = NULL
WHERE user_id = $1 AND (failed_attempts <> 0 OR locked_until IS NOT NULL)
`, userID)
	return err
}

// InsertDevice registers a family device for the parent.
func InsertDevice(ctx context.Context, pool *pgxpool.Pool, parentID uuid.UUID, tokenHash, label string) (*Device, error) {
	d := Device{ParentUserID: parentID, Label: label}
	err := pool.QueryRow(ctx, `
INSERT INTO "user".family_devices (parent_user_id, token_hash, label)
VALUES ($1, $2, $3)
RETURNING id, created_at
`, parentID, tokenHash, label).Scan(&d.ID, &d.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &d, nil
}

// ListDevices returns the parent's active family devices, newest first.
func ListDevices(ctx context.Context, pool *pgxpool.Pool, parentID uuid.UUID) ([]Device, error) {
	rows, err := pool.Query(ctx, `
SELECT id, parent_user_id, label, created_at, last_used_at
FROM "user".family_devices
WHERE parent_user_id = $1 AND revoked_at IS NULL
ORDER BY created_at DESC
`, parentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]Device, 0)
	for rows.Next() {
		var d Device
		if err := rows.Scan(&d.ID, &d.ParentUserID, &d.Label, &d.CreatedAt, &d.LastUsedAt); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

// RevokeDevice turns off a family device. Returns false when no active device matched.
func RevokeDevice(ctx context.Context, pool *pgxpool.Pool, parentID, deviceID uuid.UUID, now time.Time) (bool, error) {
	tag, err := pool.Exec(ctx, `
UPDATE "user".family_devices
SET revoked_at = $3
WHERE id = $2 AND parent_user_id = $1 AND revoked_at IS NULL
`, parentID, deviceID, now.UTC())
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}

// FindActiveDeviceByHash resolves a device secret hash to an active device, or nil.
func FindActiveDeviceByHash(ctx context.Context, pool *pgxpool.Pool, tokenHash string) (*Device, error) {
	var d Device
	err := pool.QueryRow(ctx, `
SELECT id, parent_user_id, label, created_at, last_used_at
FROM "user".family_devices
WHERE token_hash = $1 AND revoked_at IS NULL
`, tokenHash).Scan(&d.ID, &d.ParentUserID, &d.Label, &d.CreatedAt, &d.LastUsedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &d, nil
}

// TouchDevice records the latest learner sign-in on the device.
func TouchDevice(ctx context.Context, pool *pgxpool.Pool, deviceID uuid.UUID, now time.Time) error {
	_, err := pool.Exec(ctx, `UPDATE "user".family_devices SET last_used_at = $2 WHERE id = $1`, deviceID, now.UTC())
	return err
}

// ListPickerLearners returns the parent's active managed learners in the org with sign-in status.
func ListPickerLearners(ctx context.Context, pool *pgxpool.Pool, parentID, orgID uuid.UUID) ([]PickerLearner, error) {
	rows, err := pool.Query(ctx, `
SELECT u.id, COALESCE(u.display_name, ''), COALESCE(s.method, ''), s.locked_until
FROM "user".parent_student_links l
INNER JOIN "user".users u ON u.id = l.student_user_id
LEFT JOIN "user".managed_learner_sign_ins s ON s.user_id = u.id
WHERE l.parent_user_id = $1 AND l.org_id = $2
  AND l.status = 'active'
  AND u.account_type = 'managed'
  AND u.deactivated_at IS NULL
ORDER BY u.display_name NULLS LAST, l.linked_at
`, parentID, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]PickerLearner, 0)
	for rows.Next() {
		var p PickerLearner
		if err := rows.Scan(&p.ID, &p.DisplayName, &p.Method, &p.LockedUntil); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}
