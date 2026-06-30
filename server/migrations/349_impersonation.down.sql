DROP TABLE IF EXISTS auth.impersonation_tokens;
ALTER TABLE settings.platform_app_settings DROP COLUMN IF EXISTS impersonation_enabled;
