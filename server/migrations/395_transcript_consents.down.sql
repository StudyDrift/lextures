ALTER TABLE transcripts.orders
    DROP CONSTRAINT IF EXISTS orders_consent_id_fkey;

DROP INDEX IF EXISTS transcripts.idx_consents_order_active;
DROP INDEX IF EXISTS transcripts.idx_consents_user;
DROP INDEX IF EXISTS transcripts.idx_consents_order;
DROP TABLE IF EXISTS transcripts.consents;

ALTER TABLE settings.transcripts_config
    DROP COLUMN IF EXISTS consent_required;
