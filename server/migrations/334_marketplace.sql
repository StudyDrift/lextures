-- Plan 16.9 — Marketplace / Plugin System.

CREATE SCHEMA IF NOT EXISTS marketplace;

CREATE TABLE IF NOT EXISTS marketplace.apps (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    developer_user_id   UUID REFERENCES "user".users (id) ON DELETE SET NULL,
    name                TEXT NOT NULL,
    slug                TEXT NOT NULL UNIQUE,
    description         TEXT NOT NULL DEFAULT '',
    logo_url            TEXT,
    redirect_uris       TEXT[] NOT NULL DEFAULT '{}',
    requested_scopes    TEXT[] NOT NULL DEFAULT '{}',
    client_id           TEXT NOT NULL UNIQUE DEFAULT gen_random_uuid()::text,
    client_secret_hash  TEXT NOT NULL,
    client_secret_prefix TEXT NOT NULL,
    published           BOOLEAN NOT NULL DEFAULT false,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_marketplace_apps_developer
    ON marketplace.apps (developer_user_id);

CREATE INDEX IF NOT EXISTS idx_marketplace_apps_published
    ON marketplace.apps (published)
    WHERE published = true;

CREATE TABLE IF NOT EXISTS marketplace.installations (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    app_id              UUID NOT NULL REFERENCES marketplace.apps (id) ON DELETE CASCADE,
    org_id              UUID NOT NULL,
    access_token_hash   TEXT NOT NULL UNIQUE,
    access_token_prefix TEXT NOT NULL,
    refresh_token_hash  TEXT NOT NULL UNIQUE,
    refresh_token_prefix TEXT NOT NULL,
    granted_scopes      TEXT[] NOT NULL DEFAULT '{}',
    installed_by        UUID REFERENCES "user".users (id) ON DELETE SET NULL,
    installed_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    revoked_at          TIMESTAMPTZ,
    last_used_at        TIMESTAMPTZ,
    UNIQUE (app_id, org_id)
);

CREATE INDEX IF NOT EXISTS idx_marketplace_installations_org
    ON marketplace.installations (org_id)
    WHERE revoked_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_marketplace_installations_access_token
    ON marketplace.installations (access_token_hash)
    WHERE revoked_at IS NULL;

ALTER TABLE settings.platform_app_settings
    ADD COLUMN IF NOT EXISTS ff_marketplace_enabled BOOLEAN;

COMMENT ON COLUMN settings.platform_app_settings.ff_marketplace_enabled IS
    'Plan 16.9: Enables marketplace / plugin system with OAuth 2.1 app authorization (default false).';

COMMENT ON TABLE marketplace.apps IS
    'Registered third-party apps in the Lextures developer/marketplace ecosystem (plan 16.9).';

COMMENT ON TABLE marketplace.installations IS
    'Per-organisation marketplace app installations with scoped OAuth tokens (plan 16.9).';
