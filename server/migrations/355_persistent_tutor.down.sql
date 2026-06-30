-- Companion to: 355_persistent_tutor.sql

ALTER TABLE settings.platform_app_settings DROP COLUMN IF EXISTS ff_persistent_tutor;

DROP INDEX IF EXISTS course.idx_tutor_messages_session_created;
DROP TABLE IF EXISTS course.tutor_messages;

DROP INDEX IF EXISTS course.idx_tutor_sessions_student_course;
DROP TABLE IF EXISTS course.tutor_sessions;

ALTER TABLE tenant.organizations DROP COLUMN IF EXISTS tutor_session_retention_days;
ALTER TABLE "user".users DROP COLUMN IF EXISTS ai_tutor_opt_out;
