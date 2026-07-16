-- Reverse T02 transcript orders / recipients (does not restore legacy request rows).

DROP TABLE IF EXISTS transcripts.order_items;
DROP TABLE IF EXISTS transcripts.orders;
DROP TABLE IF EXISTS transcripts.recipients;

ALTER TABLE settings.transcripts_config
    DROP COLUMN IF EXISTS orders_ui_enabled;
