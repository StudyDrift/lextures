-- Plan 15.8 — Affiliate / referral / instructor revenue share.

ALTER TABLE "user".users
    ADD COLUMN IF NOT EXISTS stripe_connect_id TEXT UNIQUE;

COMMENT ON COLUMN "user".users.stripe_connect_id IS
    'Stripe Connect Express account id for creator payouts (plan 15.8).';

CREATE TABLE IF NOT EXISTS billing.creator_revenue_configs (
    user_id           UUID PRIMARY KEY REFERENCES "user".users (id) ON DELETE CASCADE,
    platform_fee_pct  NUMERIC(5, 4) NOT NULL DEFAULT 0.30,
    affiliate_fee_pct NUMERIC(5, 4) NOT NULL DEFAULT 0.10,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CHECK (platform_fee_pct >= 0 AND platform_fee_pct < 1),
    CHECK (affiliate_fee_pct >= 0 AND affiliate_fee_pct < 1)
);

COMMENT ON TABLE billing.creator_revenue_configs IS
    'Per-creator revenue share overrides (plan 15.8).';

CREATE TABLE IF NOT EXISTS billing.affiliate_codes (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID NOT NULL REFERENCES "user".users (id) ON DELETE CASCADE,
    code        TEXT NOT NULL UNIQUE,
    course_id   UUID REFERENCES course.courses (id) ON DELETE SET NULL,
    click_count INT NOT NULL DEFAULT 0,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE billing.affiliate_codes IS
    'User referral codes for affiliate commission tracking (plan 15.8).';

CREATE INDEX IF NOT EXISTS idx_billing_affiliate_codes_user
    ON billing.affiliate_codes (user_id, created_at DESC);

CREATE TABLE IF NOT EXISTS billing.earnings_ledger (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    payee_id        UUID NOT NULL REFERENCES "user".users (id) ON DELETE CASCADE,
    entry_type      TEXT NOT NULL,
    amount_cents    INT NOT NULL,
    currency        TEXT NOT NULL DEFAULT 'usd',
    stripe_event_id TEXT UNIQUE,
    stripe_transfer_id TEXT,
    course_id       UUID REFERENCES course.courses (id) ON DELETE SET NULL,
    affiliate_code  TEXT,
    status          TEXT NOT NULL DEFAULT 'pending',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CHECK (entry_type IN ('sale', 'affiliate', 'refund', 'payout')),
    CHECK (status IN ('pending', 'paid', 'held'))
);

COMMENT ON TABLE billing.earnings_ledger IS
    'Creator and affiliate earnings ledger (plan 15.8).';

CREATE INDEX IF NOT EXISTS idx_billing_earnings_payee
    ON billing.earnings_ledger (payee_id, status, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_billing_earnings_pending
    ON billing.earnings_ledger (payee_id, status)
    WHERE status = 'pending';

ALTER TABLE settings.platform_app_settings
    ADD COLUMN IF NOT EXISTS ff_revenue_share BOOLEAN;

COMMENT ON COLUMN settings.platform_app_settings.ff_revenue_share IS
    'Enables creator revenue share, affiliate tracking, and Stripe Connect payouts (plan 15.8).';
