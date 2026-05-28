// Package timezonecatalog lists IANA timezones from pg_timezone_names.
package timezonecatalog

import (
	"context"
	"sort"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Entry is one IANA timezone with a UTC offset in minutes east of UTC.
type Entry struct {
	ID            string `json:"id"`
	OffsetMinutes int    `json:"offsetMinutes"`
}

// List returns all timezone names sorted by offset then name.
func List(ctx context.Context, pool *pgxpool.Pool) ([]Entry, error) {
	rows, err := pool.Query(ctx, `
SELECT name,
       (EXTRACT(EPOCH FROM utc_offset) / 60)::int AS offset_minutes
  FROM pg_timezone_names
 ORDER BY utc_offset, name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Entry
	for rows.Next() {
		var e Entry
		if err := rows.Scan(&e.ID, &e.OffsetMinutes); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].OffsetMinutes != out[j].OffsetMinutes {
			return out[i].OffsetMinutes < out[j].OffsetMinutes
		}
		return out[i].ID < out[j].ID
	})
	return out, nil
}
