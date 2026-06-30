-- Plan 18.6: Maintenance / outage banner controls.

ALTER TABLE settings.platform_app_settings
    ADD COLUMN IF NOT EXISTS maintenance_banner_enabled BOOLEAN;

COMMENT ON COLUMN settings.platform_app_settings.maintenance_banner_enabled IS
    'Plan 18.6: Enables site-wide and org-scoped maintenance banners plus admin banner APIs.';

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'banner_severity') THEN
        CREATE TYPE platform.banner_severity AS ENUM ('info', 'warning', 'error');
    END IF;
END$$;

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'banner_scope') THEN
        CREATE TYPE platform.banner_scope AS ENUM ('global', 'org');
    END IF;
END$$;

CREATE TABLE IF NOT EXISTS platform.banners (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    scope       platform.banner_scope NOT NULL,
    org_id      UUID REFERENCES tenant.organizations (id) ON DELETE CASCADE,
    message     TEXT NOT NULL CHECK (length(message) <= 500),
    severity    platform.banner_severity NOT NULL DEFAULT 'info',
    cta_text    TEXT,
    cta_url     TEXT,
    starts_at   TIMESTAMPTZ,
    expires_at  TIMESTAMPTZ,
    is_active   BOOLEAN NOT NULL DEFAULT TRUE,
    external_id TEXT,
    created_by  UUID NOT NULL REFERENCES "user".users (id),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT banners_org_scope_check CHECK (
        (scope = 'global' AND org_id IS NULL) OR (scope = 'org' AND org_id IS NOT NULL)
    )
);

CREATE INDEX IF NOT EXISTS idx_banners_active ON platform.banners (is_active, scope, org_id);

CREATE UNIQUE INDEX IF NOT EXISTS idx_banners_external_id
    ON platform.banners (external_id)
    WHERE external_id IS NOT NULL AND external_id <> '';
