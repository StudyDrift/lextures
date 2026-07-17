-- Companion to: 407_transcript_fees.sql

DROP TABLE IF EXISTS transcripts.payment_events;
DROP INDEX IF EXISTS transcripts.idx_orders_payment_ref;

ALTER TABLE transcripts.orders
    DROP CONSTRAINT IF EXISTS orders_payment_status_check;

ALTER TABLE transcripts.orders
    DROP COLUMN IF EXISTS free_allotment_applied,
    DROP COLUMN IF EXISTS amount_refunded,
    DROP COLUMN IF EXISTS waiver_id,
    DROP COLUMN IF EXISTS payment_ref,
    DROP COLUMN IF EXISTS payment_status;

DROP TABLE IF EXISTS transcripts.waiver_applications;
DROP TABLE IF EXISTS transcripts.waiver_codes;
DROP TABLE IF EXISTS transcripts.fee_schedule;

ALTER TABLE settings.transcripts_config
    DROP COLUMN IF EXISTS fees_enabled;
