DROP TABLE IF EXISTS billing.apple_iap_transactions;

DROP INDEX IF EXISTS idx_course_apple_product_id;

ALTER TABLE course.courses
    DROP COLUMN IF EXISTS apple_product_id;

ALTER TABLE billing.user_entitlements
    DROP CONSTRAINT IF EXISTS billing_user_entitlements_acquisition_source_check;

ALTER TABLE billing.user_entitlements
    ADD CONSTRAINT billing_user_entitlements_acquisition_source_check
        CHECK (acquisition_source IN ('stripe', 'free', 'comp'));
