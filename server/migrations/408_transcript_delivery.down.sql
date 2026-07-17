-- Companion to: 408_transcript_delivery.sql

DROP TABLE IF EXISTS transcripts.postal_jobs;
DROP TABLE IF EXISTS transcripts.share_links;
DROP TABLE IF EXISTS transcripts.delivery_attempts;

ALTER TABLE transcripts.order_items
    DROP CONSTRAINT IF EXISTS order_items_delivery_method_check;

ALTER TABLE transcripts.order_items
    ADD CONSTRAINT order_items_delivery_method_check CHECK (
        delivery_method IN (
            'electronic_pesc',
            'electronic_pdf',
            'secure_link_email',
            'postal_mail',
            'api_peer'
        )
    );

ALTER TABLE transcripts.recipients
    DROP CONSTRAINT IF EXISTS recipients_capabilities_valid;

ALTER TABLE transcripts.recipients
    ADD CONSTRAINT recipients_capabilities_valid CHECK (
        capabilities <@ ARRAY[
            'electronic_pesc',
            'electronic_pdf',
            'secure_link_email',
            'postal_mail',
            'api_peer'
        ]::TEXT[]
    );

-- Strip edi_speede from capability arrays so the restored check still holds.
UPDATE transcripts.recipients
SET capabilities = array_remove(capabilities, 'edi_speede')
WHERE 'edi_speede' = ANY (capabilities);

ALTER TABLE settings.transcripts_config
    DROP COLUMN IF EXISTS delivery_v2;
