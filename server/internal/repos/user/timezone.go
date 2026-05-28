package user

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// GetTimezone returns the user's IANA timezone or nil when unset.
func GetTimezone(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID) (*string, error) {
	var tz sql.NullString
	err := pool.QueryRow(ctx, `SELECT timezone FROM "user".users WHERE id = $1`, userID).Scan(&tz)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	if !tz.Valid || tz.String == "" {
		return nil, nil
	}
	s := tz.String
	return &s, nil
}

// SetTimezone updates the user's timezone (nil clears).
func SetTimezone(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID, timezone *string) error {
	tag, err := pool.Exec(ctx, `UPDATE "user".users SET timezone = $2 WHERE id = $1`, userID, timezone)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return errors.New("user: not found for timezone update")
	}
	return nil
}

// SetTimezoneIfUnset sets timezone only when the column is currently NULL.
func SetTimezoneIfUnset(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID, timezone string) (bool, error) {
	tag, err := pool.Exec(ctx, `
UPDATE "user".users SET timezone = $2
 WHERE id = $1 AND timezone IS NULL
`, userID, timezone)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}
