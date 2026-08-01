-- MB.1: platform policy for mobile external-link handling + staged in-app browser flag.
-- Separate columns on platform_app_settings; org override in its own table (org wins).

ALTER TABLE settings.platform_app_settings
    ADD COLUMN IF NOT EXISTS mobile_link_handling TEXT NOT NULL DEFAULT 'in_app';

ALTER TABLE settings.platform_app_settings
    DROP CONSTRAINT IF EXISTS platform_app_settings_mobile_link_handling_check;

ALTER TABLE settings.platform_app_settings
    ADD CONSTRAINT platform_app_settings_mobile_link_handling_check
        CHECK (mobile_link_handling IN ('in_app', 'system', 'blocked'));

ALTER TABLE settings.platform_app_settings
    ADD COLUMN IF NOT EXISTS ff_mobile_in_app_browser BOOLEAN;

COMMENT ON COLUMN settings.platform_app_settings.mobile_link_handling IS
    'MB.1: How mobile clients open external http(s) links: in_app | system | blocked. Default in_app.';
COMMENT ON COLUMN settings.platform_app_settings.ff_mobile_in_app_browser IS
    'MB.1: Deprecated — in-app browser is always on. Kill-switch is mobile_link_handling only.';

CREATE TABLE IF NOT EXISTS tenant.org_mobile_link_handling (
    org_id UUID PRIMARY KEY REFERENCES tenant.organizations (id) ON DELETE CASCADE,
    mobile_link_handling TEXT NOT NULL
        CHECK (mobile_link_handling IN ('in_app', 'system', 'blocked')),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

COMMENT ON TABLE tenant.org_mobile_link_handling IS
    'MB.1: Org override for mobile_link_handling. When present, wins over platform default.';
