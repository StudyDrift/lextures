package user

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// DefaultLocale is the fallback BCP 47 tag when none is stored.
const DefaultLocale = "en"

// UpdateLocale sets the user's UI locale (BCP 47 tag).
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

// NormalizeLocalePrimary returns the primary BCP 47 subtag lowercased, or empty if invalid.
func NormalizeLocalePrimary(raw string) string {
	s := strings.TrimSpace(strings.ToLower(raw))
	if s == "" {
		return ""
	}
	if i := strings.IndexAny(s, "-_"); i >= 0 {
		s = s[:i]
	}
	if len(s) < 2 || len(s) > 8 {
		return ""
	}
	for _, r := range s {
		if (r < 'a' || r > 'z') && (r < '0' || r > '9') {
			return ""
		}
	}
	return s
}
