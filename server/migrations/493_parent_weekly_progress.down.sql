-- Companion to: 493_parent_weekly_progress.sql

DROP TABLE IF EXISTS settings.parent_weekly_progress_sends;

ALTER TABLE settings.notification_preferences
  DROP COLUMN IF EXISTS user_overridden;
