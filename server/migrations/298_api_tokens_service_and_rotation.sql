-- Plan 16.2 — Institutional service tokens, rotation metadata, IP hash, feature flag.

ALTER TABLE auth.api_tokens
    ALTER COLUMN owner_user_id DROP NOT NULL;

ALTER TABLE auth.api_tokens
    ADD COLUMN IF NOT EXISTS org_id UUID REFERENCES tenant.organizations (id) ON DELETE CASCADE,
    ADD COLUMN IF NOT EXISTS service_account_name TEXT,
    ADD COLUMN IF NOT EXISTS last_used_ip_hash TEXT,
    ADD COLUMN IF NOT EXISTS rotated_from_id UUID REFERENCES auth.api_tokens (id) ON DELETE SET NULL;

ALTER TABLE auth.api_tokens DROP CONSTRAINT IF EXISTS chk_api_tokens_owner;
ALTER TABLE auth.api_tokens ADD CONSTRAINT chk_api_tokens_owner CHECK (
    (owner_user_id IS NOT NULL AND org_id IS NULL) OR
    (owner_user_id IS NULL AND org_id IS NOT NULL)
);

CREATE INDEX IF NOT EXISTS idx_api_tokens_org
    ON auth.api_tokens (org_id)
    WHERE revoked_at IS NULL;

COMMENT ON COLUMN auth.api_tokens.org_id IS
    'Set for institutional service tokens (no owner_user_id).';
COMMENT ON COLUMN auth.api_tokens.service_account_name IS
    'Human-readable service account label for org-scoped tokens.';
COMMENT ON COLUMN auth.api_tokens.last_used_ip_hash IS
    'HMAC-SHA256 keyed hash of client IP (GDPR-safe; plan 16.2).';
COMMENT ON COLUMN auth.api_tokens.rotated_from_id IS
    'Previous token id when this row was created via rotation.';

ALTER TABLE settings.platform_app_settings
    ADD COLUMN IF NOT EXISTS ff_api_tokens BOOLEAN;

COMMENT ON COLUMN settings.platform_app_settings.ff_api_tokens IS
    'Enables personal and institutional API access keys (plan 16.2).';
