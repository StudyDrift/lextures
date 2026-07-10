-- Companion to: 370_feedback_submissions.sql

ALTER TABLE settings.platform_app_settings
    DROP COLUMN IF EXISTS ff_feedback;

DROP INDEX IF EXISTS feedback.idx_feedback_submissions_message_trgm;
DROP INDEX IF EXISTS feedback.uq_feedback_submissions_user_idempotency;
DROP INDEX IF EXISTS feedback.idx_feedback_submissions_user;
DROP INDEX IF EXISTS feedback.idx_feedback_submissions_created;
DROP INDEX IF EXISTS feedback.idx_feedback_submissions_org_status_created;

DROP TABLE IF EXISTS feedback.submissions;

DROP SCHEMA IF EXISTS feedback;
