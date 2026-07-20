-- Rollback companion to 430_screen_share_sessions.sql
-- See docs/runbooks/database-migration-rollback.md

ALTER TABLE settings.platform_app_settings
  DROP COLUMN IF EXISTS ff_screen_share;

ALTER TABLE course.courses
  DROP COLUMN IF EXISTS screen_share_enabled;

DROP TABLE IF EXISTS screenshare.events;
DROP TABLE IF EXISTS screenshare.participants;
DROP TABLE IF EXISTS screenshare.sessions;

DROP TYPE IF EXISTS screenshare.participant_role;
DROP TYPE IF EXISTS screenshare.present_policy;
DROP TYPE IF EXISTS screenshare.session_status;

DROP SCHEMA IF EXISTS screenshare;
