-- MKTC.7 reverse: coupon attempts + discount ceiling.

DROP INDEX IF EXISTS billing.idx_coupon_attempts_recent;
DROP INDEX IF EXISTS billing.idx_coupon_attempts_user_course;
DROP TABLE IF EXISTS billing.coupon_attempts;

ALTER TABLE settings.platform_app_settings
    DROP COLUMN IF EXISTS coupon_max_percent_off;
