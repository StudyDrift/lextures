-- T06: Electronic delivery adapters, attempts/receipts, secure share links, postal jobs.

ALTER TABLE settings.transcripts_config
    ADD COLUMN IF NOT EXISTS delivery_v2 BOOLEAN NOT NULL DEFAULT FALSE;

COMMENT ON COLUMN settings.transcripts_config.delivery_v2 IS
    'When true, full T06 adapters (PESC/EDI/PDF/postal) are enabled. When false, only api_peer (legacy webhook carrying the document) runs for ready items.';

-- Extend recipient capabilities and order_item delivery methods with edi_speede.
ALTER TABLE transcripts.recipients
    DROP CONSTRAINT IF EXISTS recipients_capabilities_valid;

ALTER TABLE transcripts.recipients
    ADD CONSTRAINT recipients_capabilities_valid CHECK (
        capabilities <@ ARRAY[
            'electronic_pesc',
            'edi_speede',
            'electronic_pdf',
            'secure_link_email',
            'postal_mail',
            'api_peer'
        ]::TEXT[]
    );

ALTER TABLE transcripts.order_items
    DROP CONSTRAINT IF EXISTS order_items_delivery_method_check;

ALTER TABLE transcripts.order_items
    ADD CONSTRAINT order_items_delivery_method_check CHECK (
        delivery_method IN (
            'electronic_pesc',
            'edi_speede',
            'electronic_pdf',
            'secure_link_email',
            'postal_mail',
            'api_peer'
        )
    );

CREATE TABLE IF NOT EXISTS transcripts.delivery_attempts (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_item_id   UUID NOT NULL REFERENCES transcripts.order_items (id) ON DELETE CASCADE,
    adapter         TEXT NOT NULL CHECK (adapter IN (
        'electronic_pesc',
        'edi_speede',
        'electronic_pdf',
        'secure_link_email',
        'postal_mail',
        'api_peer'
    )),
    attempt_no      INT  NOT NULL,
    status          TEXT NOT NULL CHECK (status IN ('queued', 'sent', 'delivered', 'opened', 'failed')),
    response_code   INT,
    detail          TEXT,
    idempotency_key TEXT NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (order_item_id, idempotency_key),
    UNIQUE (order_item_id, attempt_no)
);

COMMENT ON TABLE transcripts.delivery_attempts IS
    'Per-adapter delivery attempts and receipts for transcript order items (T06).';

CREATE INDEX IF NOT EXISTS idx_delivery_attempts_item
    ON transcripts.delivery_attempts (order_item_id, created_at);

CREATE INDEX IF NOT EXISTS idx_delivery_attempts_status
    ON transcripts.delivery_attempts (status, created_at DESC)
    WHERE status IN ('queued', 'failed');

CREATE TABLE IF NOT EXISTS transcripts.share_links (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_item_id  UUID NOT NULL REFERENCES transcripts.order_items (id) ON DELETE CASCADE,
    document_id    UUID NOT NULL REFERENCES transcripts.transcript_documents (id),
    token          TEXT NOT NULL UNIQUE,
    expires_at     TIMESTAMPTZ NOT NULL,
    max_downloads  INT NOT NULL DEFAULT 5 CHECK (max_downloads > 0),
    download_count INT NOT NULL DEFAULT 0 CHECK (download_count >= 0),
    opened_at      TIMESTAMPTZ,
    last_ip        TEXT,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE transcripts.share_links IS
    'Secure, expiring, download-capped recipient links for electronic_pdf / secure_link_email (T06).';

CREATE INDEX IF NOT EXISTS idx_share_links_item
    ON transcripts.share_links (order_item_id);

CREATE TABLE IF NOT EXISTS transcripts.postal_jobs (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_item_id  UUID NOT NULL REFERENCES transcripts.order_items (id) ON DELETE CASCADE,
    document_id    UUID NOT NULL REFERENCES transcripts.transcript_documents (id),
    address        JSONB NOT NULL,
    status         TEXT NOT NULL DEFAULT 'queued'
        CHECK (status IN ('queued', 'printing', 'shipped', 'failed', 'canceled')),
    vendor_ref     TEXT,
    detail         TEXT,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE transcripts.postal_jobs IS
    'Print/mail fulfillment queue for postal_mail delivery (T06).';

CREATE INDEX IF NOT EXISTS idx_postal_jobs_status
    ON transcripts.postal_jobs (status, created_at);

-- Seed institution recipients with EDI capability alongside PESC when present.
UPDATE transcripts.recipients
SET capabilities = CASE
    WHEN NOT ('edi_speede' = ANY (capabilities)) AND ('electronic_pesc' = ANY (capabilities))
        THEN array_append(capabilities, 'edi_speede')
    ELSE capabilities
END
WHERE type = 'institution' AND active;
