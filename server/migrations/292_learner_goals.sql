-- Plan 15.11 — Self-learner onboarding goals and diagnostic placement.

CREATE TABLE IF NOT EXISTS "user".learner_goals (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id                 UUID NOT NULL REFERENCES "user".users (id) ON DELETE CASCADE UNIQUE,
    topic                   TEXT NOT NULL DEFAULT '',
    goal_text               TEXT,
    target_date             DATE,
    daily_minutes           INT NOT NULL DEFAULT 20 CHECK (daily_minutes BETWEEN 5 AND 480),
    prior_knowledge_level   TEXT NOT NULL DEFAULT 'beginner'
        CHECK (prior_knowledge_level IN ('beginner', 'intermediate', 'advanced')),
    diagnostic_score        NUMERIC(5, 2),
    diagnostic_skipped        BOOLEAN NOT NULL DEFAULT FALSE,
    onboarding_step         INT NOT NULL DEFAULT 0,
    onboarding_completed    BOOLEAN NOT NULL DEFAULT FALSE,
    reminder_opt_in         BOOLEAN NOT NULL DEFAULT FALSE,
    reminder_time           TIME,
    recommended_course_code TEXT,
    recommended_course_title TEXT,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE "user".learner_goals IS
    'Self-learner onboarding goals, diagnostic results, and course recommendations (plan 15.11).';

CREATE INDEX IF NOT EXISTS idx_learner_goals_completed
    ON "user".learner_goals (onboarding_completed);

ALTER TABLE settings.platform_app_settings
    ADD COLUMN IF NOT EXISTS ff_onboarding_flow BOOLEAN;

COMMENT ON COLUMN settings.platform_app_settings.ff_onboarding_flow IS
    'Enables the self-learner onboarding wizard with goal capture and diagnostic placement (plan 15.11).';
