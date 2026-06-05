-- Cloud file picker OAuth / SDK credentials (plan 8.8).

ALTER TABLE settings.cloud_provider_settings
  ADD COLUMN IF NOT EXISTS client_id TEXT NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS api_key TEXT NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS app_key TEXT NOT NULL DEFAULT '';

COMMENT ON COLUMN settings.cloud_provider_settings.client_id IS 'OAuth client ID (Google Drive, OneDrive).';
COMMENT ON COLUMN settings.cloud_provider_settings.api_key IS 'Google Picker API developer key.';
COMMENT ON COLUMN settings.cloud_provider_settings.app_key IS 'Dropbox Chooser app key.';
