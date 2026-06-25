-- Plan 16.8 — Payment provider abstraction (Stripe + PayPal + iDEAL).

CREATE SCHEMA IF NOT EXISTS payments;

CREATE TABLE IF NOT EXISTS payments.transactions (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id             UUID NOT NULL REFERENCES "user".users (id) ON DELETE CASCADE,
    course_id           UUID REFERENCES course.courses (id) ON DELETE SET NULL,
    provider            TEXT NOT NULL,
    provider_txn_id     TEXT NOT NULL UNIQUE,
    idempotency_key     TEXT NOT NULL UNIQUE,
    amount_cents        INTEGER NOT NULL,
    currency            CHAR(3) NOT NULL,
    status              TEXT NOT NULL DEFAULT 'pending',
    subscription_id     TEXT,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CHECK (status IN ('pending', 'completed', 'failed', 'refunded')),
    CHECK (provider IN ('stripe', 'paypal'))
);

CREATE TABLE IF NOT EXISTS payments.subscriptions (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id             UUID NOT NULL REFERENCES "user".users (id) ON DELETE CASCADE,
    provider            TEXT NOT NULL,
    provider_sub_id     TEXT NOT NULL UNIQUE,
    plan_id             TEXT NOT NULL,
    status              TEXT NOT NULL DEFAULT 'active',
    current_period_end  TIMESTAMPTZ,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CHECK (status IN ('active', 'past_due', 'canceled')),
    CHECK (provider IN ('stripe', 'paypal'))
);

CREATE INDEX IF NOT EXISTS idx_payments_transactions_user_created
    ON payments.transactions (user_id, created_at DESC);

CREATE TABLE IF NOT EXISTS payments.webhook_jobs (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    provider            TEXT NOT NULL,
    provider_event_id   TEXT NOT NULL UNIQUE,
    payload             JSONB NOT NULL,
    headers             JSONB NOT NULL DEFAULT '{}',
    status              TEXT NOT NULL DEFAULT 'pending',
    attempts            INT NOT NULL DEFAULT 0,
    next_retry_at       TIMESTAMPTZ,
    last_error          TEXT,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    processed_at        TIMESTAMPTZ,
    CHECK (status IN ('pending', 'processing', 'completed', 'failed')),
    CHECK (provider IN ('stripe', 'paypal'))
);

CREATE INDEX IF NOT EXISTS idx_payments_webhook_jobs_due
    ON payments.webhook_jobs (status, next_retry_at)
    WHERE status IN ('pending', 'failed');

ALTER TABLE "user".users
    ADD COLUMN IF NOT EXISTS paypal_payer_id TEXT UNIQUE;

COMMENT ON COLUMN "user".users.paypal_payer_id IS
    'PayPal payer id for repeat checkout (plan 16.8).';

ALTER TABLE settings.platform_app_settings
    ADD COLUMN IF NOT EXISTS ff_payments_enabled BOOLEAN;

COMMENT ON COLUMN settings.platform_app_settings.ff_payments_enabled IS
    'Enables payment provider abstraction: multi-provider checkout, transaction history, async webhooks (plan 16.8).';

COMMENT ON TABLE payments.transactions IS
    'Financial transaction records across payment providers (plan 16.8).';

COMMENT ON TABLE payments.subscriptions IS
    'Recurring subscription records mirrored from payment providers (plan 16.8).';

COMMENT ON TABLE payments.webhook_jobs IS
    'Async payment webhook processing queue with idempotency by provider event id (plan 16.8).';
