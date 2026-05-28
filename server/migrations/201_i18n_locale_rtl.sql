-- 11.2 RTL language support: platform rtl_enabled flag (plan 11.2).
-- Depends on: 200_user_locale.sql, settings.platform_app_settings.

ALTER TABLE settings.platform_app_settings
  ADD COLUMN IF NOT EXISTS rtl_enabled BOOLEAN;

COMMENT ON COLUMN settings.platform_app_settings.rtl_enabled IS
  'When true, RTL locales mirror layout (plan 11.2); default off until audit complete.';
