-- T02: Recipient directory + multi-destination transcript orders.
-- Migrates legacy transcripts.transcript_requests into orders + order_items.

ALTER TABLE settings.transcripts_config
    ADD COLUMN IF NOT EXISTS orders_ui_enabled BOOLEAN NOT NULL DEFAULT FALSE;

COMMENT ON COLUMN settings.transcripts_config.orders_ui_enabled IS
    'When true (and ff_transcripts), students see the multi-recipient order builder. Legacy request modal remains when false.';

CREATE TABLE IF NOT EXISTS transcripts.recipients (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id        UUID REFERENCES tenant.organizations (id),
    type          TEXT NOT NULL
                    CHECK (type IN ('institution', 'application_service', 'employer', 'self', 'other')),
    name          TEXT NOT NULL,
    canonical_key TEXT,
    capabilities  TEXT[] NOT NULL DEFAULT '{}',
    email         TEXT,
    address       JSONB,
    peer_config   JSONB,
    verified      BOOLEAN NOT NULL DEFAULT FALSE,
    active        BOOLEAN NOT NULL DEFAULT TRUE,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT recipients_capabilities_valid CHECK (
        capabilities <@ ARRAY[
            'electronic_pesc',
            'electronic_pdf',
            'secure_link_email',
            'postal_mail',
            'api_peer'
        ]::TEXT[]
    )
);

COMMENT ON TABLE transcripts.recipients IS
    'Transcript receiver directory (institutions, employers, app services, self, other). NULL org_id = global/seeded.';

COMMENT ON COLUMN transcripts.recipients.canonical_key IS
    'CEEB/ACT code, domain, or normalized name used for deduplication.';

CREATE UNIQUE INDEX IF NOT EXISTS ux_recipients_canonical
    ON transcripts.recipients (
        COALESCE(org_id, '00000000-0000-0000-0000-000000000000'::uuid),
        canonical_key
    )
    WHERE canonical_key IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_recipients_search
    ON transcripts.recipients (active, type, name);

CREATE INDEX IF NOT EXISTS idx_recipients_org
    ON transcripts.recipients (org_id)
    WHERE org_id IS NOT NULL;

CREATE TABLE IF NOT EXISTS transcripts.orders (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id           UUID NOT NULL REFERENCES "user".users (id) ON DELETE CASCADE,
    org_id            UUID REFERENCES tenant.organizations (id),
    status            TEXT NOT NULL DEFAULT 'draft',
    consent_id        UUID,
    total_amount      INT,
    currency          TEXT,
    legacy_request_id UUID,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    submitted_at      TIMESTAMPTZ
);

COMMENT ON TABLE transcripts.orders IS
    'Transcript order (replaces single-destination transcript_requests). Lifecycle refined in T03.';

COMMENT ON COLUMN transcripts.orders.legacy_request_id IS
    'Provenance: source transcripts.transcript_requests.id when backfilled.';

CREATE INDEX IF NOT EXISTS idx_orders_user
    ON transcripts.orders (user_id, created_at DESC);

CREATE UNIQUE INDEX IF NOT EXISTS ux_orders_legacy_request
    ON transcripts.orders (legacy_request_id)
    WHERE legacy_request_id IS NOT NULL;

CREATE TABLE IF NOT EXISTS transcripts.order_items (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id        UUID NOT NULL REFERENCES transcripts.orders (id) ON DELETE CASCADE,
    recipient_id    UUID REFERENCES transcripts.recipients (id),
    document_id     UUID REFERENCES transcripts.transcript_documents (id),
    delivery_method TEXT NOT NULL
                      CHECK (delivery_method IN (
                          'electronic_pesc',
                          'electronic_pdf',
                          'secure_link_email',
                          'postal_mail',
                          'api_peer'
                      )),
    urgency         TEXT NOT NULL DEFAULT 'standard'
                      CHECK (urgency IN ('standard', 'rush')),
    fee_amount      INT,
    status          TEXT NOT NULL DEFAULT 'pending',
    delivered_at    TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE transcripts.order_items IS
    'One recipient × one document × one delivery method within a transcript order.';

CREATE INDEX IF NOT EXISTS idx_order_items_order
    ON transcripts.order_items (order_id);

-- Global self recipient (email / secure link).
INSERT INTO transcripts.recipients (id, org_id, type, name, canonical_key, capabilities, verified, active)
VALUES (
    'a0000000-0000-4000-8000-000000000001',
    NULL,
    'self',
    'Myself',
    'self',
    ARRAY['secure_link_email', 'electronic_pdf']::TEXT[],
    TRUE,
    TRUE
)
ON CONFLICT (id) DO NOTHING;

-- Seeded directory institutions for typeahead (global).
INSERT INTO transcripts.recipients (org_id, type, name, canonical_key, capabilities, verified, active)
VALUES
    (NULL, 'institution', 'State University', 'ceeb:001234',
     ARRAY['electronic_pesc', 'electronic_pdf', 'secure_link_email', 'postal_mail']::TEXT[], TRUE, TRUE),
    (NULL, 'institution', 'Metro Community College', 'ceeb:005678',
     ARRAY['electronic_pdf', 'secure_link_email', 'postal_mail']::TEXT[], TRUE, TRUE),
    (NULL, 'application_service', 'Common App', 'domain:commonapp.org',
     ARRAY['electronic_pdf', 'api_peer', 'secure_link_email']::TEXT[], TRUE, TRUE),
    (NULL, 'employer', 'Acme Corp Talent', 'domain:acme.example',
     ARRAY['electronic_pdf', 'secure_link_email']::TEXT[], FALSE, TRUE)
ON CONFLICT DO NOTHING;

-- Ensure ad-hoc recipients exist for legacy mail/pickup rows (dedupe by canonical key).
INSERT INTO transcripts.recipients (
    org_id, type, name, canonical_key, capabilities, email, address, verified, active
)
SELECT DISTINCT ON (ck)
    src.org_id,
    'other',
    src.recipient_name,
    src.ck,
    src.caps,
    src.delivery_email,
    src.address_json,
    FALSE,
    TRUE
FROM (
    SELECT
        r.org_id,
        r.delivery_email,
        CASE
            WHEN r.delivery_type = 'mail' THEN 'Mail recipient'
            ELSE 'Pickup'
        END AS recipient_name,
        CASE
            WHEN r.delivery_type = 'mail' THEN
                'adhoc:mail:' || md5(lower(trim(COALESCE(r.delivery_address, ''))))
            ELSE
                'adhoc:pickup:' || r.user_id::text
        END AS ck,
        CASE
            WHEN r.delivery_type = 'mail' THEN ARRAY['postal_mail']::TEXT[]
            ELSE ARRAY['secure_link_email']::TEXT[]
        END AS caps,
        CASE
            WHEN r.delivery_address IS NOT NULL AND trim(r.delivery_address) <> ''
            THEN jsonb_build_object('raw', r.delivery_address)
            ELSE NULL
        END AS address_json
    FROM transcripts.transcript_requests r
    WHERE r.delivery_type IN ('mail', 'pickup')
      AND NOT EXISTS (
          SELECT 1 FROM transcripts.orders o WHERE o.legacy_request_id = r.id
      )
) src
ORDER BY src.ck, src.org_id NULLS FIRST
ON CONFLICT DO NOTHING;

-- Backfill orders from legacy requests (one order per request).
INSERT INTO transcripts.orders (
    user_id, org_id, status, legacy_request_id, created_at, submitted_at
)
SELECT
    r.user_id,
    r.org_id,
    r.status,
    r.id,
    r.created_at,
    r.submitted_at
FROM transcripts.transcript_requests r
WHERE NOT EXISTS (
    SELECT 1 FROM transcripts.orders o WHERE o.legacy_request_id = r.id
);

-- Backfill one order_item per migrated order.
INSERT INTO transcripts.order_items (
    order_id, recipient_id, delivery_method, urgency, status, created_at
)
SELECT
    o.id,
    CASE
        WHEN r.delivery_type = 'email' THEN 'a0000000-0000-4000-8000-000000000001'::uuid
        ELSE (
            SELECT rec.id
            FROM transcripts.recipients rec
            WHERE rec.canonical_key = CASE
                WHEN r.delivery_type = 'mail' THEN
                    'adhoc:mail:' || md5(lower(trim(COALESCE(r.delivery_address, ''))))
                ELSE
                    'adhoc:pickup:' || r.user_id::text
            END
              AND COALESCE(rec.org_id, '00000000-0000-0000-0000-000000000000'::uuid)
                  = COALESCE(r.org_id, '00000000-0000-0000-0000-000000000000'::uuid)
            ORDER BY rec.created_at
            LIMIT 1
        )
    END,
    CASE r.delivery_type
        WHEN 'email' THEN 'secure_link_email'
        WHEN 'mail' THEN 'postal_mail'
        ELSE 'secure_link_email'
    END,
    CASE
        WHEN r.delivery_type = 'mail' AND COALESCE(r.urgency_days_min, r.urgency_days) <= 2 THEN 'rush'
        WHEN r.delivery_type = 'pickup' AND r.urgency_days <= 1 THEN 'rush'
        ELSE 'standard'
    END,
    CASE
        WHEN o.status = 'submitted' THEN 'delivered'
        WHEN o.status = 'failed' THEN 'failed'
        ELSE 'pending'
    END,
    r.created_at
FROM transcripts.orders o
JOIN transcripts.transcript_requests r ON r.id = o.legacy_request_id
WHERE NOT EXISTS (
    SELECT 1 FROM transcripts.order_items oi WHERE oi.order_id = o.id
);
