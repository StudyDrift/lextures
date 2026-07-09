-- Course marketplace foundation (plan MKT1).
-- Adds marketplace listing columns, free-claim entitlement support, and FFCourseMarketplace.

ALTER TABLE course.courses
    ADD COLUMN IF NOT EXISTS marketplace_listed    BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS marketplace_listed_at TIMESTAMPTZ;

COMMENT ON COLUMN course.courses.marketplace_listed IS
    'When true, the course is offered in the in-app course marketplace (plan MKT1). Independent of is_public (SEO catalog).';
COMMENT ON COLUMN course.courses.marketplace_listed_at IS
    'Set to NOW() when marketplace_listed becomes true; NULL when unlisted (plan MKT1).';

-- Storefront browse index: only listed rows (MKT3).
CREATE INDEX IF NOT EXISTS idx_courses_marketplace
    ON course.courses (marketplace_listed, catalog_category, price_cents)
    WHERE marketplace_listed = TRUE;

-- Generalize entitlements for free claims (plan MKT1).
-- stripe_event_id is already nullable (migration 279 for path bundles).
ALTER TABLE billing.user_entitlements
    ADD COLUMN IF NOT EXISTS acquisition_source TEXT NOT NULL DEFAULT 'stripe';

ALTER TABLE billing.user_entitlements
    DROP CONSTRAINT IF EXISTS billing_user_entitlements_acquisition_source_check;

ALTER TABLE billing.user_entitlements
    ADD CONSTRAINT billing_user_entitlements_acquisition_source_check
        CHECK (acquisition_source IN ('stripe', 'free', 'comp'));

COMMENT ON COLUMN billing.user_entitlements.acquisition_source IS
    'How the entitlement was granted: stripe (paid), free (claim), or comp (admin) — plan MKT1.';

-- One active course_purchase per (user, course) — supports idempotent free + paid grants.
CREATE UNIQUE INDEX IF NOT EXISTS uq_entitlement_course_per_user
    ON billing.user_entitlements (user_id, course_id)
    WHERE entitlement_type = 'course_purchase' AND status = 'active';

-- Platform flag column (default ON handled in applyPlatformBools when NULL).
ALTER TABLE settings.platform_app_settings
    ADD COLUMN IF NOT EXISTS ff_course_marketplace BOOLEAN;

COMMENT ON COLUMN settings.platform_app_settings.ff_course_marketplace IS
    'Enables the in-app course marketplace/storefront (plan MKT1). Default ON. Distinct from ff_marketplace_enabled (plugin marketplace, plan 16.9).';
