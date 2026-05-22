-- At-risk / early-warning alerts (plan 9.2).

CREATE SCHEMA IF NOT EXISTS analytics;

CREATE TABLE analytics.at_risk_config (
    org_id              UUID PRIMARY KEY REFERENCES tenant.organizations (id) ON DELETE CASCADE,
    threshold           REAL NOT NULL DEFAULT 60 CHECK (threshold BETWEEN 0 AND 100),
    weight_missing      REAL NOT NULL DEFAULT 0.35 CHECK (weight_missing BETWEEN 0 AND 1),
    weight_quiz         REAL NOT NULL DEFAULT 0.25 CHECK (weight_quiz BETWEEN 0 AND 1),
    weight_inactive     REAL NOT NULL DEFAULT 0.25 CHECK (weight_inactive BETWEEN 0 AND 1),
    weight_trend        REAL NOT NULL DEFAULT 0.15 CHECK (weight_trend BETWEEN 0 AND 1),
    quiz_avg_threshold  REAL NOT NULL DEFAULT 60 CHECK (quiz_avg_threshold BETWEEN 0 AND 100),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT at_risk_config_weights_sum CHECK (
        ABS((weight_missing + weight_quiz + weight_inactive + weight_trend) - 1.0) < 0.001
    )
);

COMMENT ON TABLE analytics.at_risk_config IS 'Per-tenant at-risk scoring threshold and signal weights (plan 9.2 FR-8).';

CREATE TABLE analytics.at_risk_scores (
    enrollment_id UUID NOT NULL REFERENCES course.course_enrollments (id) ON DELETE CASCADE,
    computed_date DATE NOT NULL,
    score         REAL NOT NULL CHECK (score BETWEEN 0 AND 100),
    missing_pct   REAL,
    quiz_avg      REAL,
    days_inactive INTEGER NOT NULL DEFAULT 0,
    grade_trend   REAL,
    top_factor    TEXT NOT NULL DEFAULT 'missing',
    PRIMARY KEY (enrollment_id, computed_date)
);

CREATE INDEX idx_at_risk_scores_date ON analytics.at_risk_scores (computed_date DESC);

CREATE TYPE analytics.alert_status AS ENUM ('active', 'dismissed', 'snoozed', 'supported', 'resolved');

CREATE TABLE analytics.at_risk_alerts (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    enrollment_id  UUID NOT NULL REFERENCES course.course_enrollments (id) ON DELETE CASCADE,
    triggered_date DATE NOT NULL,
    score          REAL NOT NULL CHECK (score BETWEEN 0 AND 100),
    status         analytics.alert_status NOT NULL DEFAULT 'active',
    top_factor     TEXT NOT NULL DEFAULT 'missing',
    snooze_until   DATE,
    notes          TEXT,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    resolved_at    TIMESTAMPTZ,
    UNIQUE (enrollment_id, triggered_date)
);

CREATE INDEX idx_at_risk_alerts_enrollment_status ON analytics.at_risk_alerts (enrollment_id, status);
CREATE INDEX idx_at_risk_alerts_course_active ON analytics.at_risk_alerts (enrollment_id)
    WHERE status IN ('active', 'snoozed', 'supported');

COMMENT ON TABLE analytics.at_risk_scores IS 'Nightly computed at-risk scores per enrollment (plan 9.2).';
COMMENT ON TABLE analytics.at_risk_alerts IS 'Instructor-facing at-risk alerts when score exceeds threshold (plan 9.2).';
