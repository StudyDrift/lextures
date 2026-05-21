-- Cloud provider file picker settings (plan 8.8): Google Drive, OneDrive, Dropbox.

-- Add cloud provider columns to existing external links table
ALTER TABLE course.module_external_links
  ADD COLUMN IF NOT EXISTS provider TEXT NOT NULL DEFAULT 'url',
  ADD COLUMN IF NOT EXISTS external_id TEXT,
  ADD COLUMN IF NOT EXISTS icon_url TEXT;

-- Admin table for enabling/disabling cloud providers per instance
CREATE SCHEMA IF NOT EXISTS settings;

CREATE TABLE IF NOT EXISTS settings.cloud_provider_settings (
    provider TEXT PRIMARY KEY,  -- 'google_drive' | 'onedrive' | 'dropbox'
    enabled BOOLEAN NOT NULL DEFAULT FALSE,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Seed defaults
INSERT INTO settings.cloud_provider_settings (provider, enabled) VALUES
    ('google_drive', FALSE),
    ('onedrive', FALSE),
    ('dropbox', FALSE)
ON CONFLICT (provider) DO NOTHING;
