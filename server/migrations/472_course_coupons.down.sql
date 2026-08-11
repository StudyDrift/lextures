-- MKTC.1 down — drop coupon ledger tables and restore acquisition_source CHECK without coupon.

DROP TABLE IF EXISTS billing.coupon_redemptions;
DROP TABLE IF EXISTS billing.course_coupons;

-- Rewrite any coupon-sourced entitlements so the narrower CHECK can be re-added.
UPDATE billing.user_entitlements
SET acquisition_source = 'free'
WHERE acquisition_source = 'coupon';

ALTER TABLE billing.user_entitlements
    DROP CONSTRAINT IF EXISTS billing_user_entitlements_acquisition_source_check;

ALTER TABLE billing.user_entitlements
    ADD CONSTRAINT billing_user_entitlements_acquisition_source_check
        CHECK (acquisition_source IN ('stripe', 'free', 'comp', 'apple'));

COMMENT ON COLUMN billing.user_entitlements.acquisition_source IS
    'How the entitlement was acquired: stripe, free, comp, or apple (StoreKit IAP).';
