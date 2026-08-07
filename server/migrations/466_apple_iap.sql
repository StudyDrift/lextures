-- Path A: Apple In-App Purchase product mapping and acquisition source (App Store 3.1.1).

ALTER TABLE billing.user_entitlements
    DROP CONSTRAINT IF EXISTS billing_user_entitlements_acquisition_source_check;

ALTER TABLE billing.user_entitlements
    ADD CONSTRAINT billing_user_entitlements_acquisition_source_check
        CHECK (acquisition_source IN ('stripe', 'free', 'comp', 'apple'));

COMMENT ON COLUMN billing.user_entitlements.acquisition_source IS
    'How the entitlement was acquired: stripe, free, comp, or apple (StoreKit IAP).';

ALTER TABLE course.courses
    ADD COLUMN IF NOT EXISTS apple_product_id TEXT;

CREATE UNIQUE INDEX IF NOT EXISTS idx_course_apple_product_id
    ON course.courses (apple_product_id)
    WHERE apple_product_id IS NOT NULL AND btrim(apple_product_id) <> '';

COMMENT ON COLUMN course.courses.apple_product_id IS
    'App Store Connect product id for this course (non-consumable). Required for iOS IAP checkout.';

CREATE TABLE IF NOT EXISTS billing.apple_iap_transactions (
    id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id               UUID NOT NULL REFERENCES "user".users (id) ON DELETE CASCADE,
    transaction_id        TEXT NOT NULL,
    original_transaction_id TEXT NOT NULL,
    product_id            TEXT NOT NULL,
    bundle_id             TEXT NOT NULL,
    environment           TEXT NOT NULL DEFAULT 'Production',
    entitlement_id        UUID REFERENCES billing.user_entitlements (id) ON DELETE SET NULL,
    signed_payload_sha256 TEXT,
    purchased_at          TIMESTAMPTZ,
    expires_at            TIMESTAMPTZ,
    raw_claims            JSONB,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (transaction_id)
);

CREATE INDEX IF NOT EXISTS idx_apple_iap_tx_user
    ON billing.apple_iap_transactions (user_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_apple_iap_tx_original
    ON billing.apple_iap_transactions (original_transaction_id);

COMMENT ON TABLE billing.apple_iap_transactions IS
    'Verified App Store Server / StoreKit 2 transactions linked to entitlements.';
