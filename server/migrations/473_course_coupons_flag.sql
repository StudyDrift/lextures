-- MKTC.2 — platform flag for creator-managed course coupon codes.

ALTER TABLE settings.platform_app_settings
    ADD COLUMN IF NOT EXISTS ff_course_coupons BOOLEAN;

COMMENT ON COLUMN settings.platform_app_settings.ff_course_coupons IS
    'Enables creator-managed course coupon codes (plan MKTC). Default OFF until GA (plan MKTC.7).';
