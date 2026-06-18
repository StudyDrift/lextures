-- Plan 15.12 — AI study buddy persistent memory and session context.

CREATE SCHEMA IF NOT EXISTS studybuddy;

CREATE TABLE IF NOT EXISTS "user".study_buddy_memory (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id             UUID NOT NULL REFERENCES "user".users (id) ON DELETE CASCADE,
    course_id           UUID NOT NULL REFERENCES course.courses (id) ON DELETE CASCADE,
    goals_summary       TEXT,
    struggle_concepts   TEXT[],
    last_session_summary TEXT,
    last_active_at      TIMESTAMPTZ,
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (user_id, course_id)
);

COMMENT ON TABLE "user".study_buddy_memory IS
    'Per-user per-course AI study buddy memory (goals, struggles, session summary; plan 15.12).';

CREATE INDEX IF NOT EXISTS idx_study_buddy_memory_user
    ON "user".study_buddy_memory (user_id);

-- Session conversation context (7-day TTL substitute for Redis; transcripts not retained long-term).
CREATE TABLE IF NOT EXISTS studybuddy.sessions (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID NOT NULL REFERENCES "user".users (id) ON DELETE CASCADE,
    course_id   UUID NOT NULL REFERENCES course.courses (id) ON DELETE CASCADE,
    messages    JSONB NOT NULL DEFAULT '[]',
    expires_at  TIMESTAMPTZ NOT NULL,
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE studybuddy.sessions IS
    'Rolling study buddy chat sessions with 7-day expiry (plan 15.12).';

CREATE INDEX IF NOT EXISTS idx_study_buddy_sessions_user_course
    ON studybuddy.sessions (user_id, course_id, updated_at DESC);

CREATE INDEX IF NOT EXISTS idx_study_buddy_sessions_expires
    ON studybuddy.sessions (expires_at);

ALTER TABLE settings.platform_app_settings
    ADD COLUMN IF NOT EXISTS ff_ai_study_buddy BOOLEAN;

COMMENT ON COLUMN settings.platform_app_settings.ff_ai_study_buddy IS
    'Enables the self-learner AI study buddy with persistent memory and proactive prompts (plan 15.12).';
