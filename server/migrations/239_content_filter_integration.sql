-- Plan 13.14: Web-content filter integration (GoGuardian, Securly).

CREATE TABLE IF NOT EXISTS tenant.content_filter_settings (
    org_id                       UUID PRIMARY KEY REFERENCES tenant.organizations (id) ON DELETE CASCADE,
    goguardian_enabled           BOOLEAN NOT NULL DEFAULT false,
    goguardian_api_key_ciphertext BYTEA,
    securly_enabled              BOOLEAN NOT NULL DEFAULT false,
    updated_at                   TIMESTAMPTZ NOT NULL DEFAULT now()
);

COMMENT ON TABLE tenant.content_filter_settings IS
    'Plan 13.14: Per-org content-filter integration settings (GoGuardian activity API, Securly catalog).';

ALTER TABLE settings.platform_app_settings
    ADD COLUMN IF NOT EXISTS ff_content_filter_integration BOOLEAN;

COMMENT ON COLUMN settings.platform_app_settings.ff_content_filter_integration IS
    'Plan 13.14: Enables web-content filter integration (allowlist, meta tags, GoGuardian/Securly).';
