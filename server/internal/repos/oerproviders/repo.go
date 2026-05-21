// Package oerproviders provides database access for OER source enablement (plan 8.9).
package oerproviders

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ProviderSetting holds the enabled state for an OER source.
type ProviderSetting struct {
	Provider  string
	Enabled   bool
	UpdatedAt time.Time
}

// List returns all OER provider settings ordered by provider name.
func List(ctx context.Context, pool *pgxpool.Pool) ([]ProviderSetting, error) {
	if pool == nil {
		return nil, errors.New("db pool is nil")
	}
	rows, err := pool.Query(ctx, `
SELECT provider, enabled, updated_at
FROM settings.oer_provider_settings
ORDER BY provider
`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ProviderSetting
	for rows.Next() {
		var ps ProviderSetting
		if err := rows.Scan(&ps.Provider, &ps.Enabled, &ps.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, ps)
	}
	return out, rows.Err()
}

// EnabledProviders returns provider IDs that are enabled.
func EnabledProviders(ctx context.Context, pool *pgxpool.Pool) ([]string, error) {
	list, err := List(ctx, pool)
	if err != nil {
		return nil, err
	}
	var out []string
	for _, p := range list {
		if p.Enabled {
			out = append(out, p.Provider)
		}
	}
	return out, nil
}

// IsEnabled reports whether a provider is enabled (unknown providers default false).
func IsEnabled(ctx context.Context, pool *pgxpool.Pool, provider string) (bool, error) {
	if pool == nil {
		return false, errors.New("db pool is nil")
	}
	var enabled bool
	err := pool.QueryRow(ctx, `
SELECT enabled FROM settings.oer_provider_settings WHERE provider = $1
`, provider).Scan(&enabled)
	if err != nil {
		return false, err
	}
	return enabled, nil
}

// SetEnabled updates the enabled flag for a provider, upserting if needed.
func SetEnabled(ctx context.Context, pool *pgxpool.Pool, provider string, enabled bool) error {
	if pool == nil {
		return errors.New("db pool is nil")
	}
	_, err := pool.Exec(ctx, `
INSERT INTO settings.oer_provider_settings (provider, enabled, updated_at)
VALUES ($1, $2, NOW())
ON CONFLICT (provider) DO UPDATE SET enabled = EXCLUDED.enabled, updated_at = NOW()
`, provider, enabled)
	return err
}
