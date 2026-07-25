-- Companion to: 439_adaptive_content_core.sql
-- See docs/runbooks/database-migration-rollback.md
-- Data-loss acknowledged: drops ACE tables and the course flag column.

DROP TABLE IF EXISTS course.adaptation_outcomes;
DROP TABLE IF EXISTS course.adaptation_servings;
DROP TABLE IF EXISTS course.content_variants;
DROP TABLE IF EXISTS course.adaptation_profiles;
DROP TABLE IF EXISTS course.adaptive_content_events;
DROP TABLE IF EXISTS course.adaptive_content_units;
DROP TABLE IF EXISTS course.adaptive_content_settings;

ALTER TABLE course.courses DROP COLUMN IF EXISTS adaptive_content_enabled;
