-- Plan 15.3 — Stripe billing, entitlements, and self-learner monetization.

CREATE SCHEMA IF NOT EXISTS billing;

ALTER TABLE "user".users
    ADD COLUMN IF NOT EXISTS stripe_customer_id TEXT UNIQUE;

COMMENT ON COLUMN "user".users.stripe_customer_id IS
    'Stripe Customer id for subscription and one-time checkout (plan 15.3).';

ALTER TABLE course.courses
    ADD COLUMN IF NOT EXISTS price_cents INT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS price_currency TEXT NOT NULL DEFAULT 'usd',
    ADD COLUMN IF NOT EXISTS freemium_free_items INT NOT NULL DEFAULT 0;

COMMENT ON COLUMN course.courses.price_cents IS
    'List price in smallest currency unit; 0 means free (plan 15.3).';
COMMENT ON COLUMN course.courses.price_currency IS
    'ISO 4217 currency for price_cents (plan 15.3).';
COMMENT ON COLUMN course.courses.freemium_free_items IS
    'Number of first module items free before payment/subscription is required (plan 15.3).';

CREATE TABLE IF NOT EXISTS billing.user_entitlements (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id           UUID NOT NULL REFERENCES "user".users (id) ON DELETE CASCADE,
    entitlement_type  TEXT NOT NULL,
    course_id         UUID REFERENCES course.courses (id) ON DELETE SET NULL,
    stripe_event_id   TEXT NOT NULL UNIQUE,
    stripe_invoice_id TEXT,
    amount_paid_cents INT NOT NULL DEFAULT 0,
    currency          TEXT NOT NULL DEFAULT 'usd',
    valid_from        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    valid_until       TIMESTAMPTZ,
    status            TEXT NOT NULL DEFAULT 'active',
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CHECK (status IN ('active', 'expired', 'refunded')),
    CHECK (entitlement_type IN (
        'course_purchase',
        'subscription_monthly',
        'subscription_annual'
    ))
);

COMMENT ON TABLE billing.user_entitlements IS
    'Paid access records derived from confirmed Stripe payments (plan 15.3).';

CREATE INDEX IF NOT EXISTS idx_billing_entitlements_user_course
    ON billing.user_entitlements (user_id, course_id, status);

CREATE INDEX IF NOT EXISTS idx_billing_entitlements_user_subscription
    ON billing.user_entitlements (user_id, entitlement_type, valid_until)
    WHERE entitlement_type LIKE 'subscription%';

ALTER TABLE settings.platform_app_settings
    ADD COLUMN IF NOT EXISTS ff_stripe_billing BOOLEAN;

COMMENT ON COLUMN settings.platform_app_settings.ff_stripe_billing IS
    'Enables Stripe checkout, subscriptions, and entitlement gating for self-learners (plan 15.3).';
