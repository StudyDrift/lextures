-- T05: Transcript fee schedules, waiver codes, payment state on orders.

ALTER TABLE settings.transcripts_config
    ADD COLUMN IF NOT EXISTS fees_enabled BOOLEAN NOT NULL DEFAULT FALSE;

COMMENT ON COLUMN settings.transcripts_config.fees_enabled IS
    'When true (and ff_transcripts), transcript orders use the org fee schedule and payment gate. When false, orders remain free.';

CREATE TABLE IF NOT EXISTS transcripts.fee_schedule (
    org_id            UUID PRIMARY KEY REFERENCES tenant.organizations (id) ON DELETE CASCADE,
    currency          TEXT NOT NULL DEFAULT 'usd',
    base_fee          INT  NOT NULL DEFAULT 0,
    rush_fee          INT  NOT NULL DEFAULT 0,
    per_recipient_fee INT  NOT NULL DEFAULT 0,
    method_surcharges JSONB NOT NULL DEFAULT '{}'::jsonb,
    free_allotment    INT  NOT NULL DEFAULT 0,
    allotment_period  TEXT NOT NULL DEFAULT 'lifetime'
        CHECK (allotment_period IN ('lifetime', 'year', 'term')),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT fee_schedule_nonneg CHECK (
        base_fee >= 0 AND rush_fee >= 0 AND per_recipient_fee >= 0 AND free_allotment >= 0
    )
);

COMMENT ON TABLE transcripts.fee_schedule IS
    'Per-org transcript fee schedule (amounts in minor units). Missing row = all zeros (free).';

COMMENT ON COLUMN transcripts.fee_schedule.method_surcharges IS
    'Map of delivery_method → surcharge in minor units, e.g. {"postal_mail": 200}.';

CREATE TABLE IF NOT EXISTS transcripts.waiver_codes (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id      UUID NOT NULL REFERENCES tenant.organizations (id) ON DELETE CASCADE,
    code        TEXT NOT NULL,
    kind        TEXT NOT NULL CHECK (kind IN ('full', 'percent', 'amount')),
    value       INT,
    max_uses    INT,
    used_count  INT NOT NULL DEFAULT 0,
    expires_at  TIMESTAMPTZ,
    created_by  UUID REFERENCES "user".users (id) ON DELETE SET NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT waiver_codes_value_ok CHECK (
        (kind = 'full' AND (value IS NULL OR value = 0))
        OR (kind = 'percent' AND value IS NOT NULL AND value >= 0 AND value <= 100)
        OR (kind = 'amount' AND value IS NOT NULL AND value >= 0)
    ),
    CONSTRAINT waiver_codes_max_uses_ok CHECK (max_uses IS NULL OR max_uses > 0),
    UNIQUE (org_id, code)
);

COMMENT ON TABLE transcripts.waiver_codes IS
    'One-time or limited-use fee waiver codes for transcript orders.';

CREATE INDEX IF NOT EXISTS idx_waiver_codes_org
    ON transcripts.waiver_codes (org_id);

CREATE TABLE IF NOT EXISTS transcripts.waiver_applications (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id        UUID NOT NULL REFERENCES transcripts.orders (id) ON DELETE CASCADE,
    org_id          UUID REFERENCES tenant.organizations (id) ON DELETE SET NULL,
    waiver_code_id  UUID REFERENCES transcripts.waiver_codes (id) ON DELETE SET NULL,
    kind            TEXT NOT NULL CHECK (kind IN ('full', 'percent', 'amount', 'admin', 'free_allotment')),
    value           INT,
    amount_waived   INT NOT NULL DEFAULT 0,
    reason          TEXT,
    applied_by      UUID REFERENCES "user".users (id) ON DELETE SET NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE transcripts.waiver_applications IS
    'Audit log of fee waivers and free-allotment applications (who/why/when).';

CREATE INDEX IF NOT EXISTS idx_waiver_applications_order
    ON transcripts.waiver_applications (order_id);

-- Payment columns on orders (expand: additive, defaults preserve free behavior).
ALTER TABLE transcripts.orders
    ADD COLUMN IF NOT EXISTS payment_status TEXT NOT NULL DEFAULT 'unpaid',
    ADD COLUMN IF NOT EXISTS payment_ref TEXT,
    ADD COLUMN IF NOT EXISTS waiver_id UUID REFERENCES transcripts.waiver_codes (id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS amount_refunded INT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS free_allotment_applied BOOLEAN NOT NULL DEFAULT FALSE;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'orders_payment_status_check'
    ) THEN
        ALTER TABLE transcripts.orders
            ADD CONSTRAINT orders_payment_status_check
            CHECK (payment_status IN (
                'unpaid', 'pending', 'paid', 'waived', 'refunded', 'partially_refunded', 'free'
            ));
    END IF;
END $$;

COMMENT ON COLUMN transcripts.orders.payment_status IS
    'T05 payment gate: unpaid|pending|paid|waived|refunded|partially_refunded|free';

COMMENT ON COLUMN transcripts.orders.payment_ref IS
    'Stripe PaymentIntent id (or Checkout Session id until PI known).';

CREATE INDEX IF NOT EXISTS idx_orders_payment_ref
    ON transcripts.orders (payment_ref)
    WHERE payment_ref IS NOT NULL;

CREATE TABLE IF NOT EXISTS transcripts.payment_events (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id        UUID NOT NULL REFERENCES transcripts.orders (id) ON DELETE CASCADE,
    stripe_event_id TEXT NOT NULL,
    event_type      TEXT NOT NULL,
    payload         JSONB,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (stripe_event_id)
);

COMMENT ON TABLE transcripts.payment_events IS
    'Idempotent Stripe webhook bookkeeping for transcript payments/refunds.';
