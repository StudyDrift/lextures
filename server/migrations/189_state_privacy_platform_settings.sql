-- Expose state_privacy_enabled in platform_app_settings so it can be toggled
-- via the admin UI without redeploying (plan 10.6).
ALTER TABLE settings.platform_app_settings
    ADD COLUMN IF NOT EXISTS state_privacy_enabled BOOLEAN;
