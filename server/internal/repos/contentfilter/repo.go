// Package contentfilter stores per-org content-filter integration settings (plan 13.14).
package contentfilter

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lextures/lextures/server/internal/crypto/appsecrets"
)

// Row is a tenant.content_filter_settings record (API key omitted in list responses).
type Row struct {
	OrgID              uuid.UUID
	GoGuardianEnabled  bool
	SecurlyEnabled     bool
	HasGoGuardianAPIKey bool
	UpdatedAt          time.Time
}

type secrets struct {
	APIKeyCiphertext []byte
}

// Get returns settings for an org, or (nil, nil) if no row exists yet.
func Get(ctx context.Context, pool *pgxpool.Pool, orgID uuid.UUID) (*Row, error) {
	row := pool.QueryRow(ctx, `
SELECT org_id, goguardian_enabled, securly_enabled,
       (goguardian_api_key_ciphertext IS NOT NULL), updated_at
FROM tenant.content_filter_settings
WHERE org_id = $1
`, orgID)
	r, err := scanPublic(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return r, err
}

// GetWithSecrets returns settings including the encrypted API key blob.
func GetWithSecrets(ctx context.Context, pool *pgxpool.Pool, orgID uuid.UUID) (*Row, *secrets, error) {
	row := pool.QueryRow(ctx, `
SELECT org_id, goguardian_enabled, securly_enabled, goguardian_api_key_ciphertext, updated_at
FROM tenant.content_filter_settings
WHERE org_id = $1
`, orgID)
	r, sec, err := scanWithSecrets(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil, nil
	}
	return r, sec, err
}

type UpsertInput struct {
	OrgID             uuid.UUID
	GoGuardianEnabled bool
	SecurlyEnabled    bool
	APIKeyPlaintext   *string
	ClearAPIKey       bool
	SecretsKey        []byte
}

// Upsert inserts or updates content-filter settings for an org.
func Upsert(ctx context.Context, pool *pgxpool.Pool, in UpsertInput) error {
	cur, curSec, err := GetWithSecrets(ctx, pool, in.OrgID)
	if err != nil {
		return err
	}
	var cipher []byte
	switch {
	case in.ClearAPIKey:
		cipher = nil
	case in.APIKeyPlaintext != nil && *in.APIKeyPlaintext != "":
		if len(in.SecretsKey) != 32 {
			return errors.New("platform secrets key not configured")
		}
		cipher, err = appsecrets.Encrypt([]byte(*in.APIKeyPlaintext), in.SecretsKey)
		if err != nil {
			return err
		}
	case cur != nil && curSec != nil:
		cipher = curSec.APIKeyCiphertext
	}
	_, err = pool.Exec(ctx, `
INSERT INTO tenant.content_filter_settings (
    org_id, goguardian_enabled, goguardian_api_key_ciphertext, securly_enabled, updated_at
) VALUES ($1, $2, $3, $4, now())
ON CONFLICT (org_id) DO UPDATE SET
    goguardian_enabled           = EXCLUDED.goguardian_enabled,
    goguardian_api_key_ciphertext = EXCLUDED.goguardian_api_key_ciphertext,
    securly_enabled              = EXCLUDED.securly_enabled,
    updated_at                   = now()
`, in.OrgID, in.GoGuardianEnabled, cipher, in.SecurlyEnabled)
	return err
}

// DecryptAPIKey returns the plaintext GoGuardian API key when configured.
func DecryptAPIKey(sec *secrets, secretsKey []byte) (string, error) {
	if sec == nil || len(sec.APIKeyCiphertext) == 0 || len(secretsKey) != 32 {
		return "", nil
	}
	plain, err := appsecrets.Decrypt(sec.APIKeyCiphertext, secretsKey)
	if err != nil {
		return "", err
	}
	return string(plain), nil
}

func scanPublic(row pgx.Row) (*Row, error) {
	var r Row
	if err := row.Scan(&r.OrgID, &r.GoGuardianEnabled, &r.SecurlyEnabled, &r.HasGoGuardianAPIKey, &r.UpdatedAt); err != nil {
		return nil, err
	}
	return &r, nil
}

func scanWithSecrets(row pgx.Row) (*Row, *secrets, error) {
	var r Row
	var sec secrets
	if err := row.Scan(&r.OrgID, &r.GoGuardianEnabled, &r.SecurlyEnabled, &sec.APIKeyCiphertext, &r.UpdatedAt); err != nil {
		return nil, nil, err
	}
	r.HasGoGuardianAPIKey = len(sec.APIKeyCiphertext) > 0
	return &r, &sec, nil
}
