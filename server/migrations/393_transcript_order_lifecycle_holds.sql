-- T03: Order lifecycle state machine, holds, audit events, auto-approval + registrar console flags.

ALTER TABLE settings.transcripts_config
    ADD COLUMN IF NOT EXISTS auto_approval_enabled BOOLEAN NOT NULL DEFAULT FALSE;

ALTER TABLE settings.transcripts_config
    ADD COLUMN IF NOT EXISTS registrar_console_enabled BOOLEAN NOT NULL DEFAULT FALSE;

COMMENT ON COLUMN settings.transcripts_config.auto_approval_enabled IS
    'When true, orders with no holds and satisfied consent/payment gates skip manual review (draft→processing).';

COMMENT ON COLUMN settings.transcripts_config.registrar_console_enabled IS
    'When true (and ff_transcripts), registrars see the fulfillment queue / holds console.';

-- Remap legacy order statuses onto the T03 lifecycle before adding CHECKs.
UPDATE transcripts.orders SET status = 'completed' WHERE status = 'submitted';
UPDATE transcripts.orders SET status = 'in_review' WHERE status = 'queued';

ALTER TABLE transcripts.orders
    DROP CONSTRAINT IF EXISTS orders_status_check;

ALTER TABLE transcripts.orders
    ADD CONSTRAINT orders_status_check CHECK (status IN (
        'draft',
        'pending_consent',
        'pending_payment',
        'in_review',
        'on_hold',
        'processing',
        'completed',
        'canceled',
        'rejected',
        'failed'
    ));

COMMENT ON COLUMN transcripts.orders.status IS
    'Order lifecycle: draft → pending_consent → pending_payment → in_review ↔ on_hold → processing → completed; terminals canceled/rejected/failed.';

UPDATE transcripts.order_items SET status = 'delivered' WHERE status = 'submitted';

ALTER TABLE transcripts.order_items
    DROP CONSTRAINT IF EXISTS order_items_status_check;

ALTER TABLE transcripts.order_items
    ADD CONSTRAINT order_items_status_check CHECK (status IN (
        'pending',
        'ready',
        'delivering',
        'delivered',
        'failed',
        'canceled'
    ));

CREATE TABLE IF NOT EXISTS transcripts.holds (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL REFERENCES "user".users (id) ON DELETE CASCADE,
    org_id          UUID REFERENCES tenant.organizations (id),
    type            TEXT NOT NULL
                      CHECK (type IN ('financial', 'disciplinary', 'registrar', 'library', 'other')),
    reason          TEXT,
    student_message TEXT,
    external_id     TEXT,
    placed_by       UUID REFERENCES "user".users (id) ON DELETE SET NULL,
    placed_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    released_by     UUID REFERENCES "user".users (id) ON DELETE SET NULL,
    released_at     TIMESTAMPTZ
);

COMMENT ON TABLE transcripts.holds IS
    'Financial/disciplinary/registrar/library holds that block official transcript issuance.';

COMMENT ON COLUMN transcripts.holds.student_message IS
    'Sanitized guidance shown to the student (never expose internal reason verbatim).';

COMMENT ON COLUMN transcripts.holds.external_id IS
    'SIS/bursar idempotency key for inbound hold upserts.';

CREATE INDEX IF NOT EXISTS idx_holds_active
    ON transcripts.holds (user_id, org_id)
    WHERE released_at IS NULL;

CREATE UNIQUE INDEX IF NOT EXISTS ux_holds_external
    ON transcripts.holds (org_id, external_id)
    WHERE external_id IS NOT NULL;

CREATE TABLE IF NOT EXISTS transcripts.order_events (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id   UUID NOT NULL REFERENCES transcripts.orders (id) ON DELETE CASCADE,
    item_id    UUID REFERENCES transcripts.order_items (id) ON DELETE CASCADE,
    from_state TEXT,
    to_state   TEXT NOT NULL,
    actor_id   UUID REFERENCES "user".users (id) ON DELETE SET NULL,
    reason     TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE transcripts.order_events IS
    'Immutable audit log of order/item state transitions (T03/T10).';

CREATE INDEX IF NOT EXISTS idx_order_events_order
    ON transcripts.order_events (order_id, created_at);

CREATE INDEX IF NOT EXISTS idx_orders_status_submitted
    ON transcripts.orders (status, submitted_at)
    WHERE status IN ('in_review', 'on_hold', 'processing', 'pending_consent', 'pending_payment');
