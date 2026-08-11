-- MKTC.1 — course-scoped coupon codes and redemption ledger.

CREATE TABLE IF NOT EXISTS billing.course_coupons (
    id                       UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    course_id                UUID NOT NULL REFERENCES course.courses (id) ON DELETE CASCADE,
    code                     TEXT NOT NULL,
    discount_type            TEXT NOT NULL,
    percent_off              NUMERIC(5,2),
    amount_off_cents         INT,
    currency                 TEXT,
    starts_at                TIMESTAMPTZ,
    ends_at                  TIMESTAMPTZ,
    max_redemptions          INT,
    max_redemptions_per_user INT NOT NULL DEFAULT 1,
    redeemed_count           INT NOT NULL DEFAULT 0,
    status                   TEXT NOT NULL DEFAULT 'active',
    note                     TEXT,
    created_by               UUID REFERENCES "user".users (id) ON DELETE SET NULL,
    created_at               TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at               TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT billing_course_coupons_code_shape_check
        CHECK (code ~ '^[A-Z0-9][A-Z0-9_-]{3,31}$'),
    CONSTRAINT billing_course_coupons_status_check
        CHECK (status IN ('active', 'disabled', 'archived')),
    CONSTRAINT billing_course_coupons_kind_check CHECK (
        (discount_type = 'percent'
             AND percent_off IS NOT NULL AND percent_off > 0 AND percent_off <= 100
             AND amount_off_cents IS NULL)
        OR (discount_type = 'fixed'
             AND amount_off_cents IS NOT NULL AND amount_off_cents > 0
             AND currency IS NOT NULL AND percent_off IS NULL)
    ),
    CONSTRAINT billing_course_coupons_window_check
        CHECK (ends_at IS NULL OR starts_at IS NULL OR ends_at > starts_at),
    CONSTRAINT billing_course_coupons_max_check
        CHECK (max_redemptions IS NULL OR max_redemptions > 0),
    CONSTRAINT billing_course_coupons_per_user_check
        CHECK (max_redemptions_per_user > 0 AND max_redemptions_per_user <= 100)
);

COMMENT ON TABLE billing.course_coupons IS
    'Creator-managed discount codes scoped to one marketplace course (plan MKTC.1).';

CREATE UNIQUE INDEX IF NOT EXISTS uq_course_coupons_code
    ON billing.course_coupons (course_id, code)
    WHERE status <> 'archived';

CREATE INDEX IF NOT EXISTS idx_course_coupons_course
    ON billing.course_coupons (course_id, status, created_at DESC);

CREATE TABLE IF NOT EXISTS billing.coupon_redemptions (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    coupon_id           UUID NOT NULL REFERENCES billing.course_coupons (id) ON DELETE CASCADE,
    course_id           UUID NOT NULL REFERENCES course.courses (id) ON DELETE CASCADE,
    user_id             UUID NOT NULL REFERENCES "user".users (id) ON DELETE CASCADE,
    entitlement_id      UUID REFERENCES billing.user_entitlements (id) ON DELETE SET NULL,
    status              TEXT NOT NULL DEFAULT 'reserved',
    checkout_session_id TEXT,
    provider_event_id   TEXT,
    list_price_cents    INT NOT NULL,
    discount_cents      INT NOT NULL,
    charged_cents       INT NOT NULL,
    currency            TEXT NOT NULL,
    reserved_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at          TIMESTAMPTZ,
    redeemed_at         TIMESTAMPTZ,
    released_at         TIMESTAMPTZ,
    CONSTRAINT billing_coupon_redemptions_status_check
        CHECK (status IN ('reserved', 'redeemed', 'released')),
    CONSTRAINT billing_coupon_redemptions_amounts_check
        CHECK (discount_cents >= 0 AND charged_cents >= 0 AND list_price_cents >= 0)
);

COMMENT ON TABLE billing.coupon_redemptions IS
    'Per-learner coupon reservations and redemptions; the authority for redemption caps (plan MKTC.1).';

CREATE UNIQUE INDEX IF NOT EXISTS uq_coupon_redemption_session
    ON billing.coupon_redemptions (checkout_session_id)
    WHERE checkout_session_id IS NOT NULL;

CREATE UNIQUE INDEX IF NOT EXISTS uq_coupon_redemption_event
    ON billing.coupon_redemptions (provider_event_id)
    WHERE provider_event_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_coupon_redemptions_coupon
    ON billing.coupon_redemptions (coupon_id, status);

CREATE INDEX IF NOT EXISTS idx_coupon_redemptions_user
    ON billing.coupon_redemptions (user_id, coupon_id, status);

-- Distinguish a 100%-off grant from a genuinely free course (plan MKTC.1 FR-14).
-- Preserve 'apple' from 466_apple_iap while adding 'coupon'.
ALTER TABLE billing.user_entitlements
    DROP CONSTRAINT IF EXISTS billing_user_entitlements_acquisition_source_check;

ALTER TABLE billing.user_entitlements
    ADD CONSTRAINT billing_user_entitlements_acquisition_source_check
        CHECK (acquisition_source IN ('stripe', 'free', 'comp', 'apple', 'coupon'));

COMMENT ON COLUMN billing.user_entitlements.acquisition_source IS
    'How the entitlement was acquired: stripe, free, comp, apple (StoreKit IAP), or coupon (100%-off grant).';
