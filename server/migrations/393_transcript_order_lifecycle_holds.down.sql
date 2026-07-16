DROP INDEX IF EXISTS transcripts.idx_orders_status_submitted;

DROP TABLE IF EXISTS transcripts.order_events;
DROP TABLE IF EXISTS transcripts.holds;

ALTER TABLE transcripts.order_items DROP CONSTRAINT IF EXISTS order_items_status_check;
ALTER TABLE transcripts.orders DROP CONSTRAINT IF EXISTS orders_status_check;

-- Best-effort reverse remap (lossy for new lifecycle states).
UPDATE transcripts.orders SET status = 'submitted' WHERE status = 'completed';
UPDATE transcripts.orders SET status = 'queued' WHERE status IN ('in_review', 'on_hold', 'processing', 'pending_consent', 'pending_payment');
UPDATE transcripts.orders SET status = 'failed' WHERE status IN ('rejected', 'canceled');

ALTER TABLE settings.transcripts_config
    DROP COLUMN IF EXISTS registrar_console_enabled;

ALTER TABLE settings.transcripts_config
    DROP COLUMN IF EXISTS auto_approval_enabled;
