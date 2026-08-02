// Package mobilelinkpolicy stores platform + org mobile link handling (MB.1).
// Columns live on settings.platform_app_settings; org overrides in tenant.org_mobile_link_handling.
// Kept separate from platformconfig's giant SELECT/UPSERT so this feature does not touch TD god-structures.
package mobilelinkpolicy

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Handling is the admin policy for external http(s) links on mobile.
type Handling string

const (
	HandlingInApp   Handling = "in_app"
	HandlingSystem  Handling = "system"
	HandlingBlocked Handling = "blocked"
)

// Platform is the effective platform-row values (before org override).
type Platform struct {
	MobileLinkHandling    Handling
	FFMobileInAppBrowser  bool
}

// Normalize maps unknown values to in_app (FR / backward compatibility).
func Normalize(raw string) Handling {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case string(HandlingSystem):
		return HandlingSystem
	case string(HandlingBlocked):
		return HandlingBlocked
	default:
		return HandlingInApp
	}
}

// IsValid reports whether raw is one of the allowed enum values.
func IsValid(raw string) bool {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case string(HandlingInApp), string(HandlingSystem), string(HandlingBlocked):
		return true
	default:
		return false
	}
}

// GetPlatform returns platform row values, or defaults when the row is missing.
// FFMobileInAppBrowser is always true (staged flag removed; use mobileLinkHandling to force system/block).
func GetPlatform(ctx context.Context, pool *pgxpool.Pool) (Platform, error) {
	out := Platform{MobileLinkHandling: HandlingInApp, FFMobileInAppBrowser: true}
	if pool == nil {
		return out, nil
	}
	var handling string
	err := pool.QueryRow(ctx, `
SELECT mobile_link_handling
FROM settings.platform_app_settings
WHERE id = 1
`).Scan(&handling)
	if errors.Is(err, pgx.ErrNoRows) {
		return out, nil
	}
	if err != nil {
		return out, err
	}
	out.MobileLinkHandling = Normalize(handling)
	out.FFMobileInAppBrowser = true
	return out, nil
}

// SetPlatform updates mobile_link_handling. browserEnabled is ignored (flag always on).
func SetPlatform(ctx context.Context, pool *pgxpool.Pool, handling *string, browserEnabled *bool) error {
	if pool == nil {
		return errors.New("database is not configured")
	}
	if handling == nil {
		return nil
	}
	// Ensure singleton row exists.
	if _, err := pool.Exec(ctx, `INSERT INTO settings.platform_app_settings (id) VALUES (1) ON CONFLICT (id) DO NOTHING`); err != nil {
		return err
	}
	if !IsValid(*handling) {
		return errors.New("mobileLinkHandling must be in_app, system, or blocked")
	}
	_, err := pool.Exec(ctx, `
UPDATE settings.platform_app_settings
SET mobile_link_handling = $1, updated_at = now()
WHERE id = 1
`, Normalize(*handling))
	return err
}

// GetOrgOverride returns the org override, or ("", false, nil) when unset.
func GetOrgOverride(ctx context.Context, pool *pgxpool.Pool, orgID uuid.UUID) (Handling, bool, error) {
	if pool == nil {
		return "", false, nil
	}
	var handling string
	err := pool.QueryRow(ctx, `
SELECT mobile_link_handling
FROM tenant.org_mobile_link_handling
WHERE org_id = $1
`, orgID).Scan(&handling)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return Normalize(handling), true, nil
}

// SetOrgOverride upserts the org override.
func SetOrgOverride(ctx context.Context, pool *pgxpool.Pool, orgID uuid.UUID, handling string) error {
	if pool == nil {
		return errors.New("database is not configured")
	}
	if !IsValid(handling) {
		return errors.New("mobileLinkHandling must be in_app, system, or blocked")
	}
	h := Normalize(handling)
	_, err := pool.Exec(ctx, `
INSERT INTO tenant.org_mobile_link_handling (org_id, mobile_link_handling, updated_at)
VALUES ($1, $2, now())
ON CONFLICT (org_id) DO UPDATE
SET mobile_link_handling = EXCLUDED.mobile_link_handling,
    updated_at = now()
`, orgID, h)
	return err
}

// ClearOrgOverride removes the org row so platform default applies.
func ClearOrgOverride(ctx context.Context, pool *pgxpool.Pool, orgID uuid.UUID) error {
	if pool == nil {
		return errors.New("database is not configured")
	}
	_, err := pool.Exec(ctx, `DELETE FROM tenant.org_mobile_link_handling WHERE org_id = $1`, orgID)
	return err
}

// Effective returns org override when set, otherwise platform handling, plus the browser flag.
func Effective(ctx context.Context, pool *pgxpool.Pool, orgID *uuid.UUID) (Handling, bool, error) {
	p, err := GetPlatform(ctx, pool)
	if err != nil {
		return HandlingInApp, false, err
	}
	if orgID != nil && pool != nil {
		if h, ok, err := GetOrgOverride(ctx, pool, *orgID); err != nil {
			return p.MobileLinkHandling, p.FFMobileInAppBrowser, err
		} else if ok {
			return h, p.FFMobileInAppBrowser, nil
		}
	}
	return p.MobileLinkHandling, p.FFMobileInAppBrowser, nil
}
