// Package navprefs manages per-user navigation personalisation (UX.7).
package navprefs

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	MaxIDs    = 64
	MaxIDLen  = 96
	MaxScope  = 128
)

// idShape: lower-kebab segments with optional dots (e.g. course.gradebook).
var idShape = regexp.MustCompile(`^[a-z0-9]+(?:[._-][a-z0-9]+)*$`)

// scopeShape mirrors the SQL CHECK constraint.
var scopeShape = regexp.MustCompile(
	`^(global|settings|admin|course:[A-Za-z0-9._~-]+|course-settings:[A-Za-z0-9._~-]+)$`,
)

// Row is one (user, scope) preference set.
type Row struct {
	Scope     string    `json:"scope"`
	Pinned    []string  `json:"pinned"`
	Hidden    []string  `json:"hidden"`
	Collapsed []string  `json:"collapsed"`
	UpdatedAt time.Time `json:"updatedAt,omitempty"`
}

// Default returns empty lists for a scope.
func Default(scope string) Row {
	return Row{
		Scope:     scope,
		Pinned:    []string{},
		Hidden:    []string{},
		Collapsed: []string{},
	}
}

// ValidScope reports whether scope matches the allowed grammar.
func ValidScope(scope string) bool {
	s := strings.TrimSpace(scope)
	if s == "" || len(s) > MaxScope {
		return false
	}
	return scopeShape.MatchString(s)
}

// ValidateIDs normalises destination/section ids; drops empty; rejects bad shape
// by omitting (unknown ids are dropped — registry is authority).
func ValidateIDs(raw []string) []string {
	if len(raw) == 0 {
		return []string{}
	}
	seen := make(map[string]struct{}, len(raw))
	out := make([]string, 0, len(raw))
	for _, r := range raw {
		id := strings.TrimSpace(r)
		if id == "" || len(id) > MaxIDLen {
			continue
		}
		if !idShape.MatchString(id) {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
		if len(out) >= MaxIDs {
			break
		}
	}
	return out
}

func scanJSONArray(src []byte) ([]string, error) {
	if len(src) == 0 {
		return []string{}, nil
	}
	var out []string
	if err := json.Unmarshal(src, &out); err != nil {
		return nil, err
	}
	if out == nil {
		return []string{}, nil
	}
	return out, nil
}

// Get returns preferences for (user, scope), or defaults when no row exists.
func Get(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID, scope string) (Row, error) {
	out := Default(scope)
	var pinned, hidden, collapsed []byte
	var updatedAt time.Time
	err := pool.QueryRow(ctx, `
SELECT pinned, hidden, collapsed, updated_at
FROM settings.user_nav_preferences
WHERE user_id = $1 AND scope = $2
`, userID, scope).Scan(&pinned, &hidden, &collapsed, &updatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return out, nil
		}
		return out, err
	}
	p, err := scanJSONArray(pinned)
	if err != nil {
		return out, err
	}
	h, err := scanJSONArray(hidden)
	if err != nil {
		return out, err
	}
	c, err := scanJSONArray(collapsed)
	if err != nil {
		return out, err
	}
	out.Pinned = p
	out.Hidden = h
	out.Collapsed = c
	out.UpdatedAt = updatedAt
	return out, nil
}

// Upsert replaces preferences for (user, scope).
func Upsert(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID, row Row) (Row, error) {
	if !ValidScope(row.Scope) {
		return Default(row.Scope), fmt.Errorf("invalid scope")
	}
	pinned := ValidateIDs(row.Pinned)
	hidden := ValidateIDs(row.Hidden)
	collapsed := ValidateIDs(row.Collapsed)
	pb, _ := json.Marshal(pinned)
	hb, _ := json.Marshal(hidden)
	cb, _ := json.Marshal(collapsed)
	_, err := pool.Exec(ctx, `
INSERT INTO settings.user_nav_preferences (user_id, scope, pinned, hidden, collapsed, updated_at)
VALUES ($1, $2, $3::jsonb, $4::jsonb, $5::jsonb, now())
ON CONFLICT (user_id, scope) DO UPDATE
    SET pinned = EXCLUDED.pinned,
        hidden = EXCLUDED.hidden,
        collapsed = EXCLUDED.collapsed,
        updated_at = now()
`, userID, row.Scope, pb, hb, cb)
	if err != nil {
		return Default(row.Scope), err
	}
	return Get(ctx, pool, userID, row.Scope)
}

// Delete removes preferences for (user, scope) — reset to defaults.
func Delete(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID, scope string) error {
	_, err := pool.Exec(ctx, `
DELETE FROM settings.user_nav_preferences
WHERE user_id = $1 AND scope = $2
`, userID, scope)
	return err
}

// GetGlobal returns the global-scope preferences (session bootstrap).
func GetGlobal(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID) (Row, error) {
	return Get(ctx, pool, userID, "global")
}
