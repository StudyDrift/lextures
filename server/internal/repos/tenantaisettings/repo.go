// Package tenantaisettings persists per-tenant AI provider configuration (plan 16.7).
package tenantaisettings

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/lextures/lextures/server/internal/crypto/appsecrets"
)

const byokSecretKey = "byok_api_key"

// Row is settings.tenant_ai_settings.
type Row struct {
	ID               uuid.UUID
	OrgID            uuid.UUID
	Provider         string
	ModelAlias       string
	FallbackProvider *string
	BYOKSecretRef    string
	Settings         map[string]any
	UpdatedBy        *uuid.UUID
	UpdatedAt        time.Time
}

// GetByOrgID returns tenant AI settings or nil when unset.
func GetByOrgID(ctx context.Context, pool *pgxpool.Pool, orgID uuid.UUID) (*Row, error) {
	if pool == nil {
		return nil, nil
	}
	row := pool.QueryRow(ctx, `
SELECT id, org_id, provider, model_alias, fallback_provider, byok_secret_ref, settings, updated_by, updated_at
  FROM settings.tenant_ai_settings
 WHERE org_id = $1
`, orgID)
	var r Row
	var settingsRaw []byte
	var fallback *string
	err := row.Scan(&r.ID, &r.OrgID, &r.Provider, &r.ModelAlias, &fallback, &r.BYOKSecretRef, &settingsRaw, &r.UpdatedBy, &r.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	r.FallbackProvider = fallback
	if len(settingsRaw) > 0 {
		_ = json.Unmarshal(settingsRaw, &r.Settings)
	}
	if r.Settings == nil {
		r.Settings = map[string]any{}
	}
	return &r, nil
}

// UpsertInput is the write payload for tenant AI settings.
type UpsertInput struct {
	Provider         string
	ModelAlias       string
	FallbackProvider *string
	BYOKSecretRef    string
	Settings         map[string]any
	UpdatedBy        uuid.UUID
}

// Upsert creates or updates tenant AI settings.
func Upsert(ctx context.Context, pool *pgxpool.Pool, orgID uuid.UUID, in UpsertInput) error {
	if pool == nil {
		return errors.New("tenantaisettings: nil pool")
	}
	settings := in.Settings
	if settings == nil {
		settings = map[string]any{}
	}
	raw, err := json.Marshal(settings)
	if err != nil {
		return err
	}
	_, err = pool.Exec(ctx, `
INSERT INTO settings.tenant_ai_settings
  (org_id, provider, model_alias, fallback_provider, byok_secret_ref, settings, updated_by, updated_at)
VALUES ($1, $2, $3, $4, NULLIF($5, ''), $6::jsonb, $7, now())
ON CONFLICT (org_id) DO UPDATE SET
  provider = EXCLUDED.provider,
  model_alias = EXCLUDED.model_alias,
  fallback_provider = EXCLUDED.fallback_provider,
  byok_secret_ref = COALESCE(NULLIF(EXCLUDED.byok_secret_ref, ''), settings.tenant_ai_settings.byok_secret_ref),
  settings = EXCLUDED.settings,
  updated_by = EXCLUDED.updated_by,
  updated_at = now()
`, orgID, in.Provider, in.ModelAlias, in.FallbackProvider, in.BYOKSecretRef, raw, in.UpdatedBy)
	return err
}

// StoreBYOK encrypts and stores a tenant API key.
func StoreBYOK(ctx context.Context, pool *pgxpool.Pool, orgID uuid.UUID, secretsKey []byte, apiKey string) error {
	if pool == nil {
		return errors.New("tenantaisettings: nil pool")
	}
	if len(secretsKey) != 32 {
		return appsecrets.ErrInvalidKey
	}
	ct, err := appsecrets.Encrypt([]byte(apiKey), secretsKey)
	if err != nil {
		return err
	}
	_, err = pool.Exec(ctx, `
INSERT INTO settings.tenant_ai_secrets (org_id, secret_key, ciphertext, updated_at)
VALUES ($1, $2, $3, now())
ON CONFLICT (org_id, secret_key) DO UPDATE SET
  ciphertext = EXCLUDED.ciphertext,
  updated_at = now()
`, orgID, byokSecretKey, ct)
	return err
}

// DecryptBYOK returns the decrypted tenant API key.
func DecryptBYOK(ctx context.Context, pool *pgxpool.Pool, orgID uuid.UUID, secretsKey []byte) (string, error) {
	if pool == nil {
		return "", errors.New("tenantaisettings: nil pool")
	}
	if len(secretsKey) != 32 {
		return "", appsecrets.ErrInvalidKey
	}
	var ct []byte
	err := pool.QueryRow(ctx, `
SELECT ciphertext FROM settings.tenant_ai_secrets
 WHERE org_id = $1 AND secret_key = $2
`, orgID, byokSecretKey).Scan(&ct)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	plain, err := appsecrets.Decrypt(ct, secretsKey)
	if err != nil {
		return "", err
	}
	return string(plain), nil
}

// BYOKConfigured reports whether a tenant has an encrypted BYOK key stored.
func BYOKConfigured(ctx context.Context, pool *pgxpool.Pool, orgID uuid.UUID) (bool, error) {
	if pool == nil {
		return false, nil
	}
	var n int
	err := pool.QueryRow(ctx, `
SELECT 1 FROM settings.tenant_ai_secrets WHERE org_id = $1 AND secret_key = $2
`, orgID, byokSecretKey).Scan(&n)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// ClearBYOK removes the legacy encrypted BYOK key and clears byok_secret_ref.
func ClearBYOK(ctx context.Context, pool *pgxpool.Pool, orgID uuid.UUID) error {
	if pool == nil {
		return errors.New("tenantaisettings: nil pool")
	}
	_, err := pool.Exec(ctx, `
DELETE FROM settings.tenant_ai_secrets WHERE org_id = $1 AND secret_key = $2
`, orgID, byokSecretKey)
	if err != nil {
		return err
	}
	_, err = pool.Exec(ctx, `
UPDATE settings.tenant_ai_settings
   SET byok_secret_ref = NULL, updated_at = now()
 WHERE org_id = $1
`, orgID)
	return err
}

// DefaultBYOKRef returns the secret ref written when BYOK is configured.
func DefaultBYOKRef() string { return byokSecretKey }