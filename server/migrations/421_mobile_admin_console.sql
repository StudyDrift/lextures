-- MOB.3: staged rollout for mobile Settings/Admin hub (grouped menu + audit log).
ALTER TABLE settings.platform_app_settings
    ADD COLUMN IF NOT EXISTS ff_mobile_admin_console BOOLEAN;

COMMENT ON COLUMN settings.platform_app_settings.ff_mobile_admin_console IS
    'MOB.3: Mobile Settings/Admin console hub (grouped menu + audit log). Default OFF.';
