// Package validation provides shared input validators.
package validation

import (
	"context"
	"strings"
	"sync"

	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	tzOnce  sync.Once
	tzValid map[string]struct{}
	tzLoad  error
)

func loadTimezones(ctx context.Context, pool *pgxpool.Pool) {
	tzOnce.Do(func() {
		rows, err := pool.Query(ctx, `SELECT name FROM pg_timezone_names`)
		if err != nil {
			tzLoad = err
			return
		}
		defer rows.Close()
		m := make(map[string]struct{})
		for rows.Next() {
			var name string
			if err := rows.Scan(&name); err != nil {
				tzLoad = err
				return
			}
			m[name] = struct{}{}
		}
		if err := rows.Err(); err != nil {
			tzLoad = err
			return
		}
		tzValid = m
	})
}

// ValidIANATimezone reports whether id is a canonical name in pg_timezone_names.
func ValidIANATimezone(ctx context.Context, pool *pgxpool.Pool, id string) (bool, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return false, nil
	}
	loadTimezones(ctx, pool)
	if tzLoad != nil {
		return false, tzLoad
	}
	_, ok := tzValid[id]
	return ok, nil
}

// NormalizeTimezone trims whitespace; empty string means clear (NULL).
func NormalizeTimezone(raw *string) *string {
	if raw == nil {
		return nil
	}
	t := strings.TrimSpace(*raw)
	if t == "" {
		return nil
	}
	return &t
}
