-- Expose gdpr_module_enabled in platform_app_settings so it can be toggled
-- via the admin UI without redeploying (plan 10.3).
ALTER TABLE settings.platform_app_settings
    ADD COLUMN IF NOT EXISTS gdpr_module_enabled BOOLEAN;
