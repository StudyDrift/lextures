-- Companion to: 368_course_marketplace.sql

DROP INDEX IF EXISTS billing.uq_entitlement_course_per_user;

ALTER TABLE billing.user_entitlements
    DROP CONSTRAINT IF EXISTS billing_user_entitlements_acquisition_source_check;

ALTER TABLE billing.user_entitlements
    DROP COLUMN IF EXISTS acquisition_source;

DROP INDEX IF EXISTS course.idx_courses_marketplace;

ALTER TABLE course.courses
    DROP COLUMN IF EXISTS marketplace_listed_at,
    DROP COLUMN IF EXISTS marketplace_listed;

ALTER TABLE settings.platform_app_settings
    DROP COLUMN IF EXISTS ff_course_marketplace;
