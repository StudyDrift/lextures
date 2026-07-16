-- AP.2 — Multi-provider credential store (platform + org BYOK).
-- Keeps legacy openrouter_api_key and tenant_ai_secrets for dual-read until AP.9.

CREATE TABLE IF NOT EXISTS settings.ai_provider_credentials (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    scope         TEXT NOT NULL CHECK (scope IN ('platform', 'org')),
    org_id        UUID REFERENCES tenant.organizations(id) ON DELETE CASCADE,
    provider      TEXT NOT NULL,
    enabled       BOOLEAN NOT NULL DEFAULT true,
    secret_ref    TEXT,
    settings      JSONB NOT NULL DEFAULT '{}',
    updated_by    UUID REFERENCES "user".users(id) ON DELETE SET NULL,
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE NULLS NOT DISTINCT (scope, org_id, provider),
    CHECK (
      (scope = 'platform' AND org_id IS NULL) OR
      (scope = 'org' AND org_id IS NOT NULL)
    )
);

CREATE INDEX IF NOT EXISTS idx_ai_provider_credentials_org
    ON settings.ai_provider_credentials (org_id)
    WHERE org_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_ai_provider_credentials_scope_provider
    ON settings.ai_provider_credentials (scope, provider);

COMMENT ON TABLE settings.ai_provider_credentials IS
    'Platform and org AI provider credentials metadata (non-secret settings); AP.2.';

CREATE TABLE IF NOT EXISTS settings.ai_provider_secrets (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    scope       TEXT NOT NULL CHECK (scope IN ('platform', 'org')),
    org_id      UUID REFERENCES tenant.organizations(id) ON DELETE CASCADE,
    provider    TEXT NOT NULL,
    secret_key  TEXT NOT NULL DEFAULT 'api_key',
    ciphertext  BYTEA NOT NULL,
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE NULLS NOT DISTINCT (scope, org_id, provider, secret_key),
    CHECK (
      (scope = 'platform' AND org_id IS NULL) OR
      (scope = 'org' AND org_id IS NOT NULL)
    )
);

CREATE INDEX IF NOT EXISTS idx_ai_provider_secrets_org
    ON settings.ai_provider_secrets (org_id)
    WHERE org_id IS NOT NULL;

COMMENT ON TABLE settings.ai_provider_secrets IS
    'AES-256-GCM encrypted AI provider API keys (PLATFORM_SECRETS_KEY); AP.2.';

-- Optional platform policy: NULL = allow tenant BYOK for all providers (FR-9).
ALTER TABLE settings.platform_app_settings
    ADD COLUMN IF NOT EXISTS ai_tenant_byok_allowed BOOLEAN;

ALTER TABLE settings.platform_app_settings
    ADD COLUMN IF NOT EXISTS ai_tenant_allowed_providers TEXT[];

COMMENT ON COLUMN settings.platform_app_settings.ai_tenant_byok_allowed IS
    'When false, tenants cannot store BYOK credentials (NULL/true = allowed); AP.2 FR-9.';

COMMENT ON COLUMN settings.platform_app_settings.ai_tenant_allowed_providers IS
    'When non-empty, tenants may only configure these provider names; NULL = all; AP.2 FR-9.';

-- Backfill org credentials + secrets from legacy tenant_ai_* tables.
INSERT INTO settings.ai_provider_secrets (scope, org_id, provider, secret_key, ciphertext, updated_at)
SELECT
    'org',
    s.org_id,
    COALESCE(NULLIF(TRIM(t.provider), ''), 'openrouter'),
    'api_key',
    s.ciphertext,
    s.updated_at
FROM settings.tenant_ai_secrets s
LEFT JOIN settings.tenant_ai_settings t ON t.org_id = s.org_id
WHERE s.secret_key = 'byok_api_key'
ON CONFLICT DO NOTHING;

INSERT INTO settings.ai_provider_credentials (
    scope, org_id, provider, enabled, secret_ref, settings, updated_by, updated_at
)
SELECT
    'org',
    t.org_id,
    COALESCE(NULLIF(TRIM(t.provider), ''), 'openrouter'),
    true,
    CASE
        WHEN EXISTS (
            SELECT 1 FROM settings.tenant_ai_secrets s
             WHERE s.org_id = t.org_id AND s.secret_key = 'byok_api_key'
        ) THEN 'api_key'
        ELSE NULL
    END,
    COALESCE(t.settings, '{}'::jsonb),
    t.updated_by,
    t.updated_at
FROM settings.tenant_ai_settings t
ON CONFLICT DO NOTHING;

-- Orgs with a BYOK secret but no tenant_ai_settings row yet.
INSERT INTO settings.ai_provider_credentials (
    scope, org_id, provider, enabled, secret_ref, settings, updated_at
)
SELECT
    'org',
    s.org_id,
    'openrouter',
    true,
    'api_key',
    '{}'::jsonb,
    s.updated_at
FROM settings.tenant_ai_secrets s
WHERE s.secret_key = 'byok_api_key'
  AND NOT EXISTS (
      SELECT 1 FROM settings.ai_provider_credentials c
       WHERE c.scope = 'org' AND c.org_id = s.org_id
  )
ON CONFLICT DO NOTHING;

-- Platform OpenRouter credential metadata when legacy plaintext key is set.
-- Ciphertext is dual-written later when PLATFORM_SECRETS_KEY is available (cannot encrypt in SQL).
INSERT INTO settings.ai_provider_credentials (
    scope, org_id, provider, enabled, secret_ref, settings, updated_at
)
SELECT
    'platform',
    NULL,
    'openrouter',
    true,
    'api_key',
    '{}'::jsonb,
    COALESCE(updated_at, now())
FROM settings.platform_app_settings
WHERE openrouter_api_key IS NOT NULL
  AND TRIM(openrouter_api_key) <> ''
ON CONFLICT DO NOTHING;
