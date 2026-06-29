-- Plan 18.1: Admin console feature flag.

ALTER TABLE settings.platform_app_settings
    ADD COLUMN IF NOT EXISTS admin_console_enabled BOOLEAN;

COMMENT ON COLUMN settings.platform_app_settings.admin_console_enabled IS
    'Plan 18.1: Enables the org admin console (/admin) and /api/v1/admin-console/* APIs.';
