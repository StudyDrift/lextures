-- Plan 18.3: Admin impersonation ("view as student") feature flag and token revocation index.

ALTER TABLE settings.platform_app_settings
    ADD COLUMN IF NOT EXISTS impersonation_enabled BOOLEAN;

COMMENT ON COLUMN settings.platform_app_settings.impersonation_enabled IS
    'Plan 18.3: Enables org admin impersonation APIs and UI (view as student).';

CREATE TABLE IF NOT EXISTS auth.impersonation_tokens (
    jti         TEXT PRIMARY KEY,
    admin_id    UUID NOT NULL REFERENCES "user".users(id),
    target_id   UUID NOT NULL REFERENCES "user".users(id),
    issued_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at  TIMESTAMPTZ NOT NULL,
    revoked_at  TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS impersonation_tokens_exp ON auth.impersonation_tokens(expires_at);

COMMENT ON TABLE auth.impersonation_tokens IS
    'Short-lived impersonation JWT revocation index (plan 18.3). Rows expire via background sweep.';
