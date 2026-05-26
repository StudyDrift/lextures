-- Expose coppa_workflow_enabled in platform_app_settings so it can be toggled
-- via the Settings → Global platform UI (plan 10.2).
ALTER TABLE settings.platform_app_settings
  ADD COLUMN IF NOT EXISTS coppa_workflow_enabled BOOLEAN;
