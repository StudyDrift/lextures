package devicepushtokens

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Row is one native device push token (APNs or FCM).
type Row struct {
	ID          uuid.UUID `json:"id"`
	UserID      uuid.UUID `json:"userId"`
	Token       string    `json:"token"`
	Platform    string    `json:"platform"`
	AppBundleID string    `json:"appBundleId,omitempty"`
	AppVersion  string    `json:"appVersion,omitempty"`
	IsActive    bool      `json:"isActive"`
	CreatedAt   time.Time `json:"createdAt"`
}

// Insert upserts a device token for the user. Returns the row id.
func Insert(
	ctx context.Context,
	pool *pgxpool.Pool,
	userID uuid.UUID,
	token, platform, appBundleID, appVersion string,
) (uuid.UUID, error) {
	var id uuid.UUID
	err := pool.QueryRow(ctx, `
INSERT INTO settings.device_push_tokens (user_id, token, platform, app_bundle_id, app_version, is_active)
VALUES ($1, $2, $3, NULLIF($4, ''), NULLIF($5, ''), true)
ON CONFLICT (user_id, token) DO UPDATE SET
  platform      = EXCLUDED.platform,
  app_bundle_id = EXCLUDED.app_bundle_id,
  app_version   = EXCLUDED.app_version,
  is_active     = true,
  last_used_at  = now()
RETURNING id
`, userID, token, platform, appBundleID, appVersion).Scan(&id)
	return id, err
}

// Delete removes a token by id and owner.
func Delete(ctx context.Context, pool *pgxpool.Pool, id, userID uuid.UUID) error {
	_, err := pool.Exec(ctx, `
DELETE FROM settings.device_push_tokens WHERE id = $1 AND user_id = $2
`, id, userID)
	return err
}

// DeleteByToken removes a token string for the owner (logout deregister).
func DeleteByToken(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID, token string) error {
	_, err := pool.Exec(ctx, `
DELETE FROM settings.device_push_tokens WHERE user_id = $1 AND token = $2
`, userID, token)
	return err
}

// ListActiveForUser returns active tokens for push fan-out.
func ListActiveForUser(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID) ([]Row, error) {
	rows, err := pool.Query(ctx, `
SELECT id, user_id, token, platform, COALESCE(app_bundle_id, ''), COALESCE(app_version, ''), is_active, created_at
FROM settings.device_push_tokens
WHERE user_id = $1 AND is_active = true
ORDER BY created_at DESC
`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Row
	for rows.Next() {
		var r Row
		if err := rows.Scan(&r.ID, &r.UserID, &r.Token, &r.Platform, &r.AppBundleID, &r.AppVersion, &r.IsActive, &r.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// ListForUser returns active tokens for session management UI.
func ListForUser(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID) ([]Row, error) {
	return ListActiveForUser(ctx, pool, userID)
}

// MarkUsed updates last_used_at after a successful delivery.
func MarkUsed(ctx context.Context, pool *pgxpool.Pool, id uuid.UUID) error {
	_, err := pool.Exec(ctx, `UPDATE settings.device_push_tokens SET last_used_at = now() WHERE id = $1`, id)
	return err
}

// MarkInactive marks a token inactive after APNs/FCM reports it invalid.
func MarkInactive(ctx context.Context, pool *pgxpool.Pool, id uuid.UUID) error {
	_, err := pool.Exec(ctx, `UPDATE settings.device_push_tokens SET is_active = false WHERE id = $1`, id)
	return err
}
