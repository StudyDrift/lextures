-- Expose ccpa_module_enabled in platform_app_settings so it can be toggled
-- via the admin UI without redeploying (plan 10.4).
ALTER TABLE settings.platform_app_settings
    ADD COLUMN IF NOT EXISTS ccpa_module_enabled BOOLEAN;
