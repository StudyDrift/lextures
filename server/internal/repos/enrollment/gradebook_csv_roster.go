package enrollment

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// EmailsByUserIDs returns trimmed emails for the given users. Missing users are omitted.
func EmailsByUserIDs(ctx context.Context, pool *pgxpool.Pool, userIDs []uuid.UUID) (map[uuid.UUID]string, error) {
	out := make(map[uuid.UUID]string)
	if len(userIDs) == 0 {
		return out, nil
	}
	rows, err := pool.Query(ctx, `
		SELECT id, email
		FROM "user".users
		WHERE id = ANY($1::uuid[])
	`, userIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var id uuid.UUID
		var email string
		if err := rows.Scan(&id, &email); err != nil {
			return nil, err
		}
		out[id] = email
	}
	return out, rows.Err()
}
