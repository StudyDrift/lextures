-- User phone number for SMS notifications; per-event SMS preference (off by default).

ALTER TABLE "user".users
  ADD COLUMN IF NOT EXISTS phone_number TEXT;

ALTER TABLE settings.notification_preferences
  ADD COLUMN IF NOT EXISTS sms_enabled BOOLEAN NOT NULL DEFAULT false;