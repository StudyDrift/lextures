-- Rollback companion to 434_modules_ai_assistant.sql
-- See docs/runbooks/database-migration-rollback.md

ALTER TABLE course.courses
  DROP COLUMN IF EXISTS modules_ai_assistant_enabled;
