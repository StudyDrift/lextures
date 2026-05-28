-- 11.2 RTL language support: user locale preference and platform rtl_enabled flag (plan 11.2).
-- Depends on: "user".users (011), settings.platform_app_settings.

ALTER TABLE "user".users
  ADD COLUMN IF NOT EXISTS locale TEXT NOT NULL DEFAULT 'en';

COMMENT ON COLUMN "user".users.locale IS
  'BCP 47 locale tag for UI language; drives document lang and RTL layout when enabled (plan 11.2).';

ALTER TABLE settings.platform_app_settings
  ADD COLUMN IF NOT EXISTS rtl_enabled BOOLEAN;

COMMENT ON COLUMN settings.platform_app_settings.rtl_enabled IS
  'When true, RTL locales mirror layout (plan 11.2); default off until audit complete.';
