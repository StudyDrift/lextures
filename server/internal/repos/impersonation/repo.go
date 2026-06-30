// Package impersonation persists impersonation JWT metadata for server-side revocation (plan 18.3).
package impersonation

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrStoreUnavailable = errors.New("impersonation: token store unavailable")

// Insert records a newly issued impersonation token.
func Insert(ctx context.Context, pool *pgxpool.Pool, jti string, adminID, targetID uuid.UUID, expiresAt time.Time) error {
	if pool == nil {
		return ErrStoreUnavailable
	}
	_, err := pool.Exec(ctx, `
INSERT INTO auth.impersonation_tokens (jti, admin_id, target_id, expires_at)
VALUES ($1, $2, $3, $4)
`, jti, adminID, targetID, expiresAt.UTC())
	return err
}

// IsActive reports whether jti exists, is not revoked, and has not expired.
func IsActive(ctx context.Context, pool *pgxpool.Pool, jti string, now time.Time) (bool, error) {
	if pool == nil {
		return false, ErrStoreUnavailable
	}
	var revokedAt *time.Time
	var expiresAt time.Time
	err := pool.QueryRow(ctx, `
SELECT expires_at, revoked_at FROM auth.impersonation_tokens WHERE jti = $1
`, jti).Scan(&expiresAt, &revokedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if revokedAt != nil {
		return false, nil
	}
	return expiresAt.After(now.UTC()), nil
}

// Revoke marks an impersonation token as revoked.
func Revoke(ctx context.Context, pool *pgxpool.Pool, jti string, now time.Time) error {
	if pool == nil {
		return ErrStoreUnavailable
	}
	_, err := pool.Exec(ctx, `
UPDATE auth.impersonation_tokens SET revoked_at = $2 WHERE jti = $1 AND revoked_at IS NULL
`, jti, now.UTC())
	return err
}
