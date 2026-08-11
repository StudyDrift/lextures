-- MKTC.2 — drop platform flag for course coupons.

ALTER TABLE settings.platform_app_settings
    DROP COLUMN IF EXISTS ff_course_coupons;
