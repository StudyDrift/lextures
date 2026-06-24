-- Per-tenant AI provider configuration and encrypted BYOK storage (plan 16.7).

CREATE TABLE IF NOT EXISTS settings.tenant_ai_settings (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id            UUID NOT NULL UNIQUE REFERENCES tenant.organizations(id) ON DELETE CASCADE,
    provider          TEXT NOT NULL DEFAULT 'openrouter',
    model_alias       TEXT NOT NULL DEFAULT 'claude-3-5-sonnet',
    fallback_provider TEXT,
    byok_secret_ref   TEXT,
    settings          JSONB NOT NULL DEFAULT '{}',
    updated_by        UUID REFERENCES "user".users(id) ON DELETE SET NULL,
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_tenant_ai_settings_provider
    ON settings.tenant_ai_settings (provider);

COMMENT ON TABLE settings.tenant_ai_settings IS
    'Per-tenant AI provider, model alias, fallback chain, and non-secret provider settings (plan 16.7).';

CREATE TABLE IF NOT EXISTS settings.tenant_ai_secrets (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id      UUID NOT NULL REFERENCES tenant.organizations(id) ON DELETE CASCADE,
    secret_key  TEXT NOT NULL,
    ciphertext  BYTEA NOT NULL,
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (org_id, secret_key)
);

COMMENT ON TABLE settings.tenant_ai_secrets IS
    'AES-256-GCM encrypted tenant BYOK API keys; referenced by tenant_ai_settings.byok_secret_ref (plan 16.7).';

ALTER TABLE analytics.ai_usage_log
    ADD COLUMN IF NOT EXISTS provider TEXT NOT NULL DEFAULT 'openrouter';

COMMENT ON COLUMN analytics.ai_usage_log.provider IS
    'AI provider backend that served the request (openrouter, anthropic, openai, bedrock, vertex).';