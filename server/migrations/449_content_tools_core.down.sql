-- Companion to: 449_content_tools_core.sql
-- See docs/runbooks/database-migration-rollback.md
-- Data-loss acknowledged: drops Content Tools tables and the course flag column.

DROP TABLE IF EXISTS course.content_tool_events;
DROP TABLE IF EXISTS course.content_tool_states;
DROP TABLE IF EXISTS course.content_tool_instances;
DROP TABLE IF EXISTS course.content_tool_settings;

ALTER TABLE course.courses DROP COLUMN IF EXISTS content_tools_enabled;
