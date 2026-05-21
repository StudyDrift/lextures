// Package cloudproviders provides database access for cloud file picker settings (plan 8.8).
package cloudproviders

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ProviderSetting holds the enabled/disabled state for a cloud storage provider.
type ProviderSetting struct {
	Provider  string
	Enabled   bool
	UpdatedAt time.Time
}

// List returns all cloud provider settings ordered by provider name.
func List(ctx context.Context, pool *pgxpool.Pool) ([]ProviderSetting, error) {
	if pool == nil {
		return nil, errors.New("db pool is nil")
	}
	rows, err := pool.Query(ctx, `
SELECT provider, enabled, updated_at
FROM settings.cloud_provider_settings
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

// SetEnabled updates the enabled flag for a provider, upserting if needed.
func SetEnabled(ctx context.Context, pool *pgxpool.Pool, provider string, enabled bool) error {
	if pool == nil {
		return errors.New("db pool is nil")
	}
	_, err := pool.Exec(ctx, `
INSERT INTO settings.cloud_provider_settings (provider, enabled, updated_at)
VALUES ($1, $2, NOW())
ON CONFLICT (provider) DO UPDATE SET enabled = EXCLUDED.enabled, updated_at = NOW()
`, provider, enabled)
	return err
}
