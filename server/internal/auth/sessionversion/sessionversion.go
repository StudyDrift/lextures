// Package sessionversion reads and bumps users.jwt_session_version for JWT invalidation.
package sessionversion

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Read returns users.jwt_session_version for API JWT validation.
func Read(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID) (int64, error) {
	var v int64
	err := pool.QueryRow(ctx, `SELECT jwt_session_version FROM "user".users WHERE id = $1`, userID).Scan(&v)
	return v, err
}
