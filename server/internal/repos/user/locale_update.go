package user

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// UpdateLocale sets the user locale preference.
func UpdateLocale(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID, locale string) (*Row, error) {
	const q = `UPDATE "user".users
SET locale = $2
WHERE id = $1
RETURNING ` + userRowColumns
	row := pool.QueryRow(ctx, q, userID, locale)
	r, err := scanInsertedUserRow(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return r, nil
}
