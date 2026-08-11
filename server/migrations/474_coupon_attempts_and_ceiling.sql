-- MKTC.7 — coupon attempt audit trail + platform discount ceiling.

CREATE TABLE IF NOT EXISTS billing.coupon_attempts (
    id         BIGSERIAL PRIMARY KEY,
    user_id    UUID REFERENCES "user".users (id) ON DELETE CASCADE,
    course_id  UUID REFERENCES course.courses (id) ON DELETE CASCADE,
    code_hash  TEXT NOT NULL,          -- salted hash; raw code never stored for unknown codes
    reason     TEXT NOT NULL,
    ip_prefix  TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_coupon_attempts_user_course
    ON billing.coupon_attempts (user_id, course_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_coupon_attempts_recent
    ON billing.coupon_attempts (created_at DESC);

COMMENT ON TABLE billing.coupon_attempts IS
    'Bounded 30-day log of failed coupon applications for enumeration detection (plan MKTC.7).';

ALTER TABLE settings.platform_app_settings
    ADD COLUMN IF NOT EXISTS coupon_max_percent_off NUMERIC(5,2);

COMMENT ON COLUMN settings.platform_app_settings.coupon_max_percent_off IS
    'Optional cap on creator coupon percent discounts; NULL/100 = uncapped (plan MKTC.7).';
