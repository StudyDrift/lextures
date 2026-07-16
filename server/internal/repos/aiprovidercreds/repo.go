// Package aiprovidercreds persists platform and org AI provider credentials (plan AP.2).
package aiprovidercreds

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/lextures/lextures/server/internal/crypto/appsecrets"
)

const (
	ScopePlatform = "platform"
	ScopeOrg      = "org"

	// SecretKeyAPIKey is the primary secret_key for provider API keys.
	SecretKeyAPIKey = "api_key"
	// SecretKeyAWSAccessKeyID is used for Bedrock auth_mode=access_key.
	SecretKeyAWSAccessKeyID = "aws_access_key_id"
	// SecretKeyAWSSecretAccessKey is used for Bedrock auth_mode=access_key.
	SecretKeyAWSSecretAccessKey = "aws_secret_access_key"
	// SecretKeyServiceAccountJSON is used for Vertex auth_mode=service_account.
	SecretKeyServiceAccountJSON = "service_account_json"

	// MaxServiceAccountJSONBytes caps uploaded GCP service account JSON (AP.8).
	MaxServiceAccountJSONBytes = 64 << 10 // 64 KiB
)

// KnownSecretKeys lists all secret_key values the store accepts (AP.8 FR-6).
var KnownSecretKeys = []string{
	SecretKeyAPIKey,
	SecretKeyAWSAccessKeyID,
	SecretKeyAWSSecretAccessKey,
	SecretKeyServiceAccountJSON,
}

// Credential is one settings.ai_provider_credentials row plus whether a secret exists.
type Credential struct {
	ID             uuid.UUID
	Scope          string
	OrgID          *uuid.UUID
	Provider       string
	Enabled        bool
	SecretRef      string
	Settings       map[string]any
	UpdatedBy      *uuid.UUID
	UpdatedAt      time.Time
	SecretConfigured bool
	// SecretsConfigured maps secret_key → present (never includes plaintext).
	SecretsConfigured map[string]bool
}

// UpsertInput is the write payload for credential metadata (and optional secret).
type UpsertInput struct {
	Enabled   *bool
	Settings  map[string]any
	UpdatedBy *uuid.UUID
	// SetSettings when true replaces settings even if Settings is empty/nil.
	SetSettings bool
}

// ListByScope returns credentials for platform or a specific org.
func ListByScope(ctx context.Context, pool *pgxpool.Pool, scope string, orgID *uuid.UUID) ([]Credential, error) {
	if pool == nil {
		return nil, errors.New("aiprovidercreds: nil pool")
	}
	scope = strings.TrimSpace(scope)
	if err := validateScope(scope, orgID); err != nil {
		return nil, err
	}
	rows, err := pool.Query(ctx, `
SELECT c.id, c.scope, c.org_id, c.provider, c.enabled, COALESCE(c.secret_ref, ''), c.settings,
       c.updated_by, c.updated_at,
       EXISTS (
         SELECT 1 FROM settings.ai_provider_secrets s
          WHERE s.scope = c.scope
            AND s.provider = c.provider
            AND s.secret_key = $3
            AND ((c.org_id IS NULL AND s.org_id IS NULL) OR s.org_id = c.org_id)
       ) AS secret_configured
  FROM settings.ai_provider_credentials c
 WHERE c.scope = $1
   AND (($2::uuid IS NULL AND c.org_id IS NULL) OR c.org_id = $2)
 ORDER BY c.provider
`, scope, orgID, SecretKeyAPIKey)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Credential
	for rows.Next() {
		c, err := scanCredential(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for i := range out {
		keys, err := ListConfiguredSecretKeys(ctx, pool, scope, orgID, out[i].Provider)
		if err != nil {
			return nil, err
		}
		out[i].SecretsConfigured = secretKeysMap(keys)
	}
	return out, nil
}

// Get returns one credential or nil when unset.
func Get(ctx context.Context, pool *pgxpool.Pool, scope string, orgID *uuid.UUID, provider string) (*Credential, error) {
	if pool == nil {
		return nil, errors.New("aiprovidercreds: nil pool")
	}
	provider = strings.TrimSpace(provider)
	if provider == "" {
		return nil, errors.New("aiprovidercreds: empty provider")
	}
	if err := validateScope(scope, orgID); err != nil {
		return nil, err
	}
	row := pool.QueryRow(ctx, `
SELECT c.id, c.scope, c.org_id, c.provider, c.enabled, COALESCE(c.secret_ref, ''), c.settings,
       c.updated_by, c.updated_at,
       EXISTS (
         SELECT 1 FROM settings.ai_provider_secrets s
          WHERE s.scope = c.scope
            AND s.provider = c.provider
            AND s.secret_key = $4
            AND ((c.org_id IS NULL AND s.org_id IS NULL) OR s.org_id = c.org_id)
       ) AS secret_configured
  FROM settings.ai_provider_credentials c
 WHERE c.scope = $1
   AND (($2::uuid IS NULL AND c.org_id IS NULL) OR c.org_id = $2)
   AND c.provider = $3
`, scope, orgID, provider, SecretKeyAPIKey)
	c, err := scanCredential(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	keys, err := ListConfiguredSecretKeys(ctx, pool, scope, orgID, provider)
	if err != nil {
		return nil, err
	}
	c.SecretsConfigured = secretKeysMap(keys)
	return &c, nil
}

type scannable interface {
	Scan(dest ...any) error
}

func scanCredential(row scannable) (Credential, error) {
	var c Credential
	var settingsRaw []byte
	err := row.Scan(
		&c.ID, &c.Scope, &c.OrgID, &c.Provider, &c.Enabled, &c.SecretRef, &settingsRaw,
		&c.UpdatedBy, &c.UpdatedAt, &c.SecretConfigured,
	)
	if err != nil {
		return Credential{}, err
	}
	if len(settingsRaw) > 0 {
		_ = json.Unmarshal(settingsRaw, &c.Settings)
	}
	if c.Settings == nil {
		c.Settings = map[string]any{}
	}
	return c, nil
}

// Upsert creates or updates credential metadata. Does not clear secrets.
func Upsert(ctx context.Context, pool *pgxpool.Pool, scope string, orgID *uuid.UUID, provider string, in UpsertInput) error {
	if pool == nil {
		return errors.New("aiprovidercreds: nil pool")
	}
	provider = strings.TrimSpace(provider)
	if provider == "" {
		return errors.New("aiprovidercreds: empty provider")
	}
	if err := validateScope(scope, orgID); err != nil {
		return err
	}
	enabled := true
	enabledSet := in.Enabled != nil
	if enabledSet {
		enabled = *in.Enabled
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
INSERT INTO settings.ai_provider_credentials
  (scope, org_id, provider, enabled, settings, updated_by, updated_at)
VALUES ($1, $2, $3, $4, $5::jsonb, $6, now())
ON CONFLICT (scope, org_id, provider) DO UPDATE SET
  enabled = CASE WHEN $7::boolean THEN EXCLUDED.enabled ELSE settings.ai_provider_credentials.enabled END,
  settings = CASE WHEN $8::boolean THEN EXCLUDED.settings ELSE settings.ai_provider_credentials.settings END,
  updated_by = COALESCE(EXCLUDED.updated_by, settings.ai_provider_credentials.updated_by),
  updated_at = now()
`, scope, orgID, provider, enabled, raw, in.UpdatedBy, enabledSet, in.SetSettings || in.Settings != nil)
	return err
}

// MarkSecretRef sets secret_ref after a successful StoreSecret.
func MarkSecretRef(ctx context.Context, pool *pgxpool.Pool, scope string, orgID *uuid.UUID, provider, secretRef string) error {
	if pool == nil {
		return errors.New("aiprovidercreds: nil pool")
	}
	_, err := pool.Exec(ctx, `
UPDATE settings.ai_provider_credentials
   SET secret_ref = NULLIF($4, ''), updated_at = now()
 WHERE scope = $1
   AND (($2::uuid IS NULL AND org_id IS NULL) OR org_id = $2)
   AND provider = $3
`, scope, orgID, provider, secretRef)
	return err
}

// StoreSecret encrypts and stores the primary API key for the provider.
func StoreSecret(ctx context.Context, pool *pgxpool.Pool, scope string, orgID *uuid.UUID, provider string, secretsKey []byte, apiKey string) error {
	return StoreSecretKeyed(ctx, pool, scope, orgID, provider, SecretKeyAPIKey, secretsKey, apiKey)
}

// StoreSecretKeyed encrypts and stores a named secret for the provider (AP.8 FR-6).
func StoreSecretKeyed(ctx context.Context, pool *pgxpool.Pool, scope string, orgID *uuid.UUID, provider, secretKey string, secretsKey []byte, plaintext string) error {
	if pool == nil {
		return errors.New("aiprovidercreds: nil pool")
	}
	if len(secretsKey) != 32 {
		return appsecrets.ErrInvalidKey
	}
	provider = strings.TrimSpace(provider)
	if provider == "" {
		return errors.New("aiprovidercreds: empty provider")
	}
	secretKey = strings.TrimSpace(secretKey)
	if !IsKnownSecretKey(secretKey) {
		return fmt.Errorf("aiprovidercreds: unknown secret_key %q", secretKey)
	}
	if err := validateScope(scope, orgID); err != nil {
		return err
	}
	if secretKey == SecretKeyServiceAccountJSON && len(plaintext) > MaxServiceAccountJSONBytes {
		return fmt.Errorf("aiprovidercreds: service_account_json exceeds %d bytes", MaxServiceAccountJSONBytes)
	}
	ct, err := appsecrets.Encrypt([]byte(plaintext), secretsKey)
	if err != nil {
		return err
	}
	_, err = pool.Exec(ctx, `
INSERT INTO settings.ai_provider_secrets (scope, org_id, provider, secret_key, ciphertext, updated_at)
VALUES ($1, $2, $3, $4, $5, now())
ON CONFLICT (scope, org_id, provider, secret_key) DO UPDATE SET
  ciphertext = EXCLUDED.ciphertext,
  updated_at = now()
`, scope, orgID, provider, secretKey, ct)
	if err != nil {
		return err
	}
	if secretKey == SecretKeyAPIKey {
		return MarkSecretRef(ctx, pool, scope, orgID, provider, SecretKeyAPIKey)
	}
	return nil
}

// DecryptSecret returns the decrypted API key, or empty when unset.
func DecryptSecret(ctx context.Context, pool *pgxpool.Pool, scope string, orgID *uuid.UUID, provider string, secretsKey []byte) (string, error) {
	return DecryptSecretKeyed(ctx, pool, scope, orgID, provider, SecretKeyAPIKey, secretsKey)
}

// DecryptSecretKeyed returns a named decrypted secret, or empty when unset.
func DecryptSecretKeyed(ctx context.Context, pool *pgxpool.Pool, scope string, orgID *uuid.UUID, provider, secretKey string, secretsKey []byte) (string, error) {
	if pool == nil {
		return "", errors.New("aiprovidercreds: nil pool")
	}
	if len(secretsKey) != 32 {
		return "", appsecrets.ErrInvalidKey
	}
	secretKey = strings.TrimSpace(secretKey)
	if secretKey == "" {
		secretKey = SecretKeyAPIKey
	}
	var ct []byte
	err := pool.QueryRow(ctx, `
SELECT ciphertext FROM settings.ai_provider_secrets
 WHERE scope = $1
   AND (($2::uuid IS NULL AND org_id IS NULL) OR org_id = $2)
   AND provider = $3
   AND secret_key = $4
`, scope, orgID, provider, secretKey).Scan(&ct)
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

// DecryptAllSecrets returns all configured secrets for a provider (AP.8 FR-6).
// Callers must not log or return plaintext values.
func DecryptAllSecrets(ctx context.Context, pool *pgxpool.Pool, scope string, orgID *uuid.UUID, provider string, secretsKey []byte) (map[string]string, error) {
	out := map[string]string{}
	if pool == nil || len(secretsKey) != 32 {
		return out, nil
	}
	keys, err := ListConfiguredSecretKeys(ctx, pool, scope, orgID, provider)
	if err != nil {
		return nil, err
	}
	for _, k := range keys {
		v, err := DecryptSecretKeyed(ctx, pool, scope, orgID, provider, k, secretsKey)
		if err != nil {
			return nil, fmt.Errorf("aiprovidercreds: decrypt %s/%s/%s: %w", scope, provider, k, err)
		}
		if v != "" {
			out[k] = v
		}
	}
	return out, nil
}

// SecretConfigured reports whether the primary API key exists for the provider.
func SecretConfigured(ctx context.Context, pool *pgxpool.Pool, scope string, orgID *uuid.UUID, provider string) (bool, error) {
	return SecretKeyedConfigured(ctx, pool, scope, orgID, provider, SecretKeyAPIKey)
}

// SecretKeyedConfigured reports whether a named secret exists.
func SecretKeyedConfigured(ctx context.Context, pool *pgxpool.Pool, scope string, orgID *uuid.UUID, provider, secretKey string) (bool, error) {
	if pool == nil {
		return false, nil
	}
	secretKey = strings.TrimSpace(secretKey)
	if secretKey == "" {
		secretKey = SecretKeyAPIKey
	}
	var n int
	err := pool.QueryRow(ctx, `
SELECT 1 FROM settings.ai_provider_secrets
 WHERE scope = $1
   AND (($2::uuid IS NULL AND org_id IS NULL) OR org_id = $2)
   AND provider = $3
   AND secret_key = $4
`, scope, orgID, provider, secretKey).Scan(&n)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// ListConfiguredSecretKeys returns secret_key names present for the provider (no plaintext).
func ListConfiguredSecretKeys(ctx context.Context, pool *pgxpool.Pool, scope string, orgID *uuid.UUID, provider string) ([]string, error) {
	if pool == nil {
		return nil, nil
	}
	rows, err := pool.Query(ctx, `
SELECT secret_key FROM settings.ai_provider_secrets
 WHERE scope = $1
   AND (($2::uuid IS NULL AND org_id IS NULL) OR org_id = $2)
   AND provider = $3
 ORDER BY secret_key
`, scope, orgID, provider)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var k string
		if err := rows.Scan(&k); err != nil {
			return nil, err
		}
		out = append(out, k)
	}
	return out, rows.Err()
}

// ClearSecret deletes the primary API key and clears secret_ref. Credential row remains.
func ClearSecret(ctx context.Context, pool *pgxpool.Pool, scope string, orgID *uuid.UUID, provider string) error {
	return ClearSecretKeyed(ctx, pool, scope, orgID, provider, SecretKeyAPIKey)
}

// ClearSecretKeyed deletes a named secret. Clears secret_ref when clearing api_key.
func ClearSecretKeyed(ctx context.Context, pool *pgxpool.Pool, scope string, orgID *uuid.UUID, provider, secretKey string) error {
	if pool == nil {
		return errors.New("aiprovidercreds: nil pool")
	}
	secretKey = strings.TrimSpace(secretKey)
	if secretKey == "" {
		secretKey = SecretKeyAPIKey
	}
	_, err := pool.Exec(ctx, `
DELETE FROM settings.ai_provider_secrets
 WHERE scope = $1
   AND (($2::uuid IS NULL AND org_id IS NULL) OR org_id = $2)
   AND provider = $3
   AND secret_key = $4
`, scope, orgID, provider, secretKey)
	if err != nil {
		return err
	}
	if secretKey == SecretKeyAPIKey {
		return MarkSecretRef(ctx, pool, scope, orgID, provider, "")
	}
	return nil
}

// IsKnownSecretKey reports whether secretKey is an allowed material type.
func IsKnownSecretKey(secretKey string) bool {
	for _, k := range KnownSecretKeys {
		if k == secretKey {
			return true
		}
	}
	return false
}

func secretKeysMap(keys []string) map[string]bool {
	out := map[string]bool{}
	for _, k := range keys {
		out[k] = true
	}
	return out
}

// Delete removes credential metadata and secrets for a provider.
func Delete(ctx context.Context, pool *pgxpool.Pool, scope string, orgID *uuid.UUID, provider string) error {
	if pool == nil {
		return errors.New("aiprovidercreds: nil pool")
	}
	_, err := pool.Exec(ctx, `
DELETE FROM settings.ai_provider_secrets
 WHERE scope = $1
   AND (($2::uuid IS NULL AND org_id IS NULL) OR org_id = $2)
   AND provider = $3
`, scope, orgID, provider)
	if err != nil {
		return err
	}
	_, err = pool.Exec(ctx, `
DELETE FROM settings.ai_provider_credentials
 WHERE scope = $1
   AND (($2::uuid IS NULL AND org_id IS NULL) OR org_id = $2)
   AND provider = $3
`, scope, orgID, provider)
	return err
}

// ResolveAPIKey implements FR-5 dual-read for a single scope:
// encrypted store first, then optional legacyOpenRouter for platform openrouter.
func ResolveAPIKey(
	ctx context.Context,
	pool *pgxpool.Pool,
	scope string,
	orgID *uuid.UUID,
	provider string,
	secretsKey []byte,
	legacyOpenRouter string,
) (key string, settings map[string]any, enabled bool, err error) {
	cred, err := Get(ctx, pool, scope, orgID, provider)
	if err != nil {
		return "", nil, false, err
	}
	if cred != nil && !cred.Enabled {
		return "", cred.Settings, false, nil
	}
	settings = map[string]any{}
	if cred != nil {
		settings = cred.Settings
		enabled = cred.Enabled
	}
	if len(secretsKey) == 32 {
		key, err = DecryptSecret(ctx, pool, scope, orgID, provider, secretsKey)
		if err != nil {
			// Fail closed for this provider (decrypt error).
			return "", settings, enabled, fmt.Errorf("aiprovidercreds: decrypt %s/%s: %w", scope, provider, err)
		}
		if key != "" {
			return key, settings, true, nil
		}
	}
	if scope == ScopePlatform && provider == "openrouter" {
		legacy := strings.TrimSpace(legacyOpenRouter)
		if legacy != "" {
			return legacy, settings, true, nil
		}
	}
	if cred != nil {
		return "", settings, cred.Enabled, nil
	}
	return "", nil, false, nil
}

func validateScope(scope string, orgID *uuid.UUID) error {
	switch scope {
	case ScopePlatform:
		if orgID != nil {
			return errors.New("aiprovidercreds: platform scope must have nil org_id")
		}
		return nil
	case ScopeOrg:
		if orgID == nil {
			return errors.New("aiprovidercreds: org scope requires org_id")
		}
		return nil
	default:
		return fmt.Errorf("aiprovidercreds: invalid scope %q", scope)
	}
}
