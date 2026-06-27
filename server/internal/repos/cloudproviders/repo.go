// Package cloudproviders provides database access for cloud file picker settings (plan 8.8).
package cloudproviders

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ProviderSetting holds enablement and SDK credentials for a cloud storage provider.
type ProviderSetting struct {
	Provider  string
	Enabled   bool
	ClientID  string
	APIKey    string
	AppKey    string
	UpdatedAt time.Time
}

// ProviderUpdate holds optional fields for updating a provider row.
type ProviderUpdate struct {
	Enabled  *bool
	ClientID *string
	APIKey   *string
	AppKey   *string
}

// List returns all cloud provider settings ordered by provider name.
func List(ctx context.Context, pool *pgxpool.Pool) ([]ProviderSetting, error) {
	if pool == nil {
		return nil, errors.New("db pool is nil")
	}
	rows, err := pool.Query(ctx, `
SELECT provider, enabled, client_id, api_key, app_key, updated_at
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
		if err := rows.Scan(&ps.Provider, &ps.Enabled, &ps.ClientID, &ps.APIKey, &ps.AppKey, &ps.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, ps)
	}
	return out, rows.Err()
}

// IsConfigured reports whether a provider has the credentials required for its picker.
func IsConfigured(ps ProviderSetting) bool {
	switch ps.Provider {
	case "google_drive":
		return strings.TrimSpace(ps.ClientID) != "" && strings.TrimSpace(ps.APIKey) != ""
	case "onedrive":
		return strings.TrimSpace(ps.ClientID) != ""
	case "dropbox":
		return strings.TrimSpace(ps.AppKey) != ""
	default:
		return false
	}
}

// EnabledConfigured returns enabled providers that have required credentials.
func EnabledConfigured(ctx context.Context, pool *pgxpool.Pool) ([]ProviderSetting, error) {
	list, err := List(ctx, pool)
	if err != nil {
		return nil, err
	}
	var out []ProviderSetting
	for _, ps := range list {
		if ps.Enabled && IsConfigured(ps) {
			out = append(out, ps)
		}
	}
	return out, nil
}

// Update applies partial updates to a provider row, upserting if needed.
func Update(ctx context.Context, pool *pgxpool.Pool, provider string, upd ProviderUpdate) error {
	if pool == nil {
		return errors.New("db pool is nil")
	}
	enabled := false
	if upd.Enabled != nil {
		enabled = *upd.Enabled
	}
	clientID := ""
	if upd.ClientID != nil {
		clientID = strings.TrimSpace(*upd.ClientID)
	}
	apiKey := ""
	if upd.APIKey != nil {
		apiKey = strings.TrimSpace(*upd.APIKey)
	}
	appKey := ""
	if upd.AppKey != nil {
		appKey = strings.TrimSpace(*upd.AppKey)
	}

	// Preserve existing values when a field is omitted.
	var existing ProviderSetting
	err := pool.QueryRow(ctx, `
SELECT provider, enabled, client_id, api_key, app_key, updated_at
FROM settings.cloud_provider_settings
WHERE provider = $1
`, provider).Scan(&existing.Provider, &existing.Enabled, &existing.ClientID, &existing.APIKey, &existing.AppKey, &existing.UpdatedAt)
	if err == nil {
		if upd.Enabled == nil {
			enabled = existing.Enabled
		}
		if upd.ClientID == nil {
			clientID = existing.ClientID
		}
		if upd.APIKey == nil {
			apiKey = existing.APIKey
		}
		if upd.AppKey == nil {
			appKey = existing.AppKey
		}
	}

	_, err = pool.Exec(ctx, `
INSERT INTO settings.cloud_provider_settings (provider, enabled, client_id, api_key, app_key, updated_at)
VALUES ($1, $2, $3, $4, $5, NOW())
ON CONFLICT (provider) DO UPDATE SET
  enabled = EXCLUDED.enabled,
  client_id = EXCLUDED.client_id,
  api_key = EXCLUDED.api_key,
  app_key = EXCLUDED.app_key,
  updated_at = NOW()
`, provider, enabled, clientID, apiKey, appKey)
	return err
}
