-- Per-course at-risk scoring overrides (plan 9.2).

ALTER TABLE analytics.at_risk_config
    ADD COLUMN IF NOT EXISTS inactive_days_threshold INTEGER NOT NULL DEFAULT 7
        CHECK (inactive_days_threshold BETWEEN 1 AND 90),
    ADD COLUMN IF NOT EXISTS missing_pct_threshold REAL NOT NULL DEFAULT 100
        CHECK (missing_pct_threshold BETWEEN 1 AND 100);

CREATE TABLE IF NOT EXISTS analytics.course_at_risk_config (
    course_id               UUID PRIMARY KEY REFERENCES course.courses (id) ON DELETE CASCADE,
    threshold               REAL NOT NULL DEFAULT 60 CHECK (threshold BETWEEN 0 AND 100),
    weight_missing          REAL NOT NULL DEFAULT 0.35 CHECK (weight_missing BETWEEN 0 AND 1),
    weight_quiz             REAL NOT NULL DEFAULT 0.25 CHECK (weight_quiz BETWEEN 0 AND 1),
    weight_inactive         REAL NOT NULL DEFAULT 0.25 CHECK (weight_inactive BETWEEN 0 AND 1),
    weight_trend            REAL NOT NULL DEFAULT 0.15 CHECK (weight_trend BETWEEN 0 AND 1),
    quiz_avg_threshold      REAL NOT NULL DEFAULT 60 CHECK (quiz_avg_threshold BETWEEN 0 AND 100),
    inactive_days_threshold INTEGER NOT NULL DEFAULT 7 CHECK (inactive_days_threshold BETWEEN 1 AND 90),
    missing_pct_threshold   REAL NOT NULL DEFAULT 100 CHECK (missing_pct_threshold BETWEEN 1 AND 100),
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT course_at_risk_config_weights_sum CHECK (
        ABS((weight_missing + weight_quiz + weight_inactive + weight_trend) - 1.0) < 0.001
    )
);

COMMENT ON TABLE analytics.course_at_risk_config IS
    'Per-course at-risk scoring thresholds and signal weights; overrides tenant defaults when present.';
