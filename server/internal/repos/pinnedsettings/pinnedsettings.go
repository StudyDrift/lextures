// Package pinnedsettings manages per-user ordered pinned setting keys for editor panels (plan PS.2).
package pinnedsettings

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Surfaces supported by the pinned-settings API.
const (
	SurfaceAssignment = "assignment"
	SurfaceQuiz       = "quiz"
)

// MaxPins is the hard cap on keys per surface (schema CHECK + API).
const MaxPins = 12

// MaxKeyLen is the max length of a single setting key.
const MaxKeyLen = 96

// keyShape matches PS.1/PS.2: lower-kebab segments joined by . or - only after normalisation.
var keyShape = regexp.MustCompile(`^[a-z0-9]+(?:[.-][a-z0-9]+)*$`)

// Surface is a validated editor surface name.
type Surface string

// Surfaces returns the ordered list of known surfaces (for GET defaults).
func Surfaces() []string {
	return []string{SurfaceAssignment, SurfaceQuiz}
}

// ValidSurface reports whether s is an allowed surface path segment.
func ValidSurface(s string) bool {
	switch s {
	case SurfaceAssignment, SurfaceQuiz:
		return true
	default:
		return false
	}
}

// Row is one (user, surface) pin list.
type Row struct {
	Surface     string    `json:"surface"`
	SettingKeys []string  `json:"settingKeys"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

// All is the GET response shape: every surface, empty when unset.
type All struct {
	Assignment []string `json:"assignment"`
	Quiz       []string `json:"quiz"`
}

// DefaultAll returns empty pin lists for every surface.
func DefaultAll() All {
	return All{
		Assignment: []string{},
		Quiz:       []string{},
	}
}

// GetAll returns pins for all surfaces for the user. Missing rows become empty arrays.
func GetAll(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID) (All, error) {
	out := DefaultAll()
	rows, err := pool.Query(ctx, `
SELECT surface, setting_keys, updated_at
FROM settings.user_pinned_settings
WHERE user_id = $1
`, userID)
	if err != nil {
		return out, err
	}
	defer rows.Close()
	for rows.Next() {
		var surface string
		var keys []string
		var updatedAt time.Time
		if err := rows.Scan(&surface, &keys, &updatedAt); err != nil {
			return out, err
		}
		if keys == nil {
			keys = []string{}
		}
		switch surface {
		case SurfaceAssignment:
			out.Assignment = keys
		case SurfaceQuiz:
			out.Quiz = keys
		}
	}
	return out, rows.Err()
}

// Get returns the pin list for one surface, or empty when no row exists.
func Get(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID, surface string) ([]string, error) {
	var keys []string
	err := pool.QueryRow(ctx, `
SELECT setting_keys
FROM settings.user_pinned_settings
WHERE user_id = $1 AND surface = $2
`, userID, surface).Scan(&keys)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return []string{}, nil
		}
		return nil, err
	}
	if keys == nil {
		return []string{}, nil
	}
	return keys, nil
}

// RejectReason classifies validation failures for metrics / logs.
type RejectReason string

const (
	ReasonShape      RejectReason = "shape"
	ReasonTooMany    RejectReason = "too_many"
	ReasonDuplicate  RejectReason = "duplicate"
	ReasonBadSurface RejectReason = "bad_surface"
)

// ValidateKeys normalises and validates an ordered list of setting keys.
// On success it returns the normalised keys (trim + lowercase). On failure it
// returns a non-empty RejectReason and a human-readable error.
func ValidateKeys(keys []string) (normalised []string, reason RejectReason, err error) {
	if len(keys) > MaxPins {
		return nil, ReasonTooMany, fmt.Errorf("at most %d pinned settings are allowed", MaxPins)
	}
	seen := make(map[string]struct{}, len(keys))
	out := make([]string, 0, len(keys))
	for _, raw := range keys {
		k := strings.ToLower(strings.TrimSpace(raw))
		if k == "" || len(k) > MaxKeyLen || !keyShape.MatchString(k) {
			return nil, ReasonShape, fmt.Errorf("invalid setting key %q", raw)
		}
		if _, dup := seen[k]; dup {
			return nil, ReasonDuplicate, fmt.Errorf("duplicate setting key %q", k)
		}
		seen[k] = struct{}{}
		out = append(out, k)
	}
	return out, "", nil
}

// Upsert replaces the pin list for (user, surface) and returns the full All snapshot.
// Empty keys clear the surface (row is retained with '{}').
func Upsert(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID, surface string, keys []string) (All, error) {
	if keys == nil {
		keys = []string{}
	}
	_, err := pool.Exec(ctx, `
INSERT INTO settings.user_pinned_settings (user_id, surface, setting_keys, updated_at)
VALUES ($1, $2, $3, now())
ON CONFLICT (user_id, surface) DO UPDATE
    SET setting_keys = EXCLUDED.setting_keys,
        updated_at   = now()
`, userID, surface, keys)
	if err != nil {
		return DefaultAll(), err
	}
	return GetAll(ctx, pool, userID)
}

// ListForExport returns all pin rows for a user (DSAR / access export).
func ListForExport(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID) ([]Row, error) {
	rows, err := pool.Query(ctx, `
SELECT surface, setting_keys, updated_at
FROM settings.user_pinned_settings
WHERE user_id = $1
ORDER BY surface
`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Row
	for rows.Next() {
		var r Row
		if err := rows.Scan(&r.Surface, &r.SettingKeys, &r.UpdatedAt); err != nil {
			return nil, err
		}
		if r.SettingKeys == nil {
			r.SettingKeys = []string{}
		}
		out = append(out, r)
	}
	return out, rows.Err()
}
