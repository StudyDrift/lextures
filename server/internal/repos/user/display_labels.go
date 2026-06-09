package user

import (
	"context"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// DisplayLabel returns a human-readable label for a user row.
func DisplayLabel(displayName *string, email string) string {
	if displayName != nil {
		if s := strings.TrimSpace(*displayName); s != "" {
			return s
		}
	}
	return strings.TrimSpace(email)
}

// DisplayLabelsByIDs returns display labels keyed by user id (display_name or email fallback).
func DisplayLabelsByIDs(ctx context.Context, pool *pgxpool.Pool, ids []uuid.UUID) (map[uuid.UUID]string, error) {
	out := make(map[uuid.UUID]string, len(ids))
	if len(ids) == 0 {
		return out, nil
	}
	unique := make([]uuid.UUID, 0, len(ids))
	seen := make(map[uuid.UUID]struct{}, len(ids))
	for _, id := range ids {
		if id == uuid.Nil {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		unique = append(unique, id)
	}
	if len(unique) == 0 {
		return out, nil
	}
	rows, err := pool.Query(ctx, `
SELECT id, COALESCE(NULLIF(TRIM(display_name), ''), email) AS display_label
FROM "user".users
WHERE id = ANY($1)
`, unique)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var id uuid.UUID
		var label string
		if err := rows.Scan(&id, &label); err != nil {
			return nil, err
		}
		out[id] = label
	}
	return out, rows.Err()
}
