package apitokens

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// DeleteExpiredAndRevoked removes API tokens that are past their expiry or have
// been revoked, returning the number deleted. Run by the expired_token_cleanup
// scheduled job to keep auth.api_tokens from accumulating dead rows
// (plan 17.4 FR-4, AC-3).
func DeleteExpiredAndRevoked(ctx context.Context, pool *pgxpool.Pool, now time.Time) (int64, error) {
	tag, err := pool.Exec(ctx, `
DELETE FROM auth.api_tokens
WHERE (expires_at IS NOT NULL AND expires_at < $1)
   OR revoked_at IS NOT NULL
`, now)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}
