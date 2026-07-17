DELETE FROM settings.email_template_slots
WHERE id IN ('transcript_order_update', 'transcript_order_exception');

DROP INDEX IF EXISTS transcripts.notification_log_order_idx;
DROP INDEX IF EXISTS transcripts.notification_log_idempotency;
DROP TABLE IF EXISTS transcripts.notification_log;
