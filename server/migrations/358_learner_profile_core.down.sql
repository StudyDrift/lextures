DROP SCHEMA IF EXISTS learner CASCADE;

ALTER TABLE settings.platform_app_settings DROP COLUMN IF EXISTS learner_profile_enabled;