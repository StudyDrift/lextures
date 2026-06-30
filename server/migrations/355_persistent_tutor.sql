-- Plan 19.1 — Persistent AI tutor across sessions.

ALTER TABLE "user".users
    ADD COLUMN IF NOT EXISTS ai_tutor_opt_out BOOLEAN NOT NULL DEFAULT FALSE;

COMMENT ON COLUMN "user".users.ai_tutor_opt_out IS
    'When true, the student cannot use the AI tutor (plan 19.1 FR-5).';

ALTER TABLE tenant.organizations
    ADD COLUMN IF NOT EXISTS tutor_session_retention_days INTEGER NOT NULL DEFAULT 365;

COMMENT ON COLUMN tenant.organizations.tutor_session_retention_days IS
    'Days to retain tutor session messages before nightly purge (plan 19.1 FR-7).';

CREATE TABLE IF NOT EXISTS course.tutor_sessions (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    student_id  UUID NOT NULL REFERENCES "user".users (id) ON DELETE CASCADE,
    course_id   UUID NOT NULL REFERENCES course.courses (id) ON DELETE CASCADE,
    title       TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_active TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE course.tutor_sessions IS
    'Named AI tutor conversation sessions per student per course (plan 19.1).';

CREATE INDEX IF NOT EXISTS idx_tutor_sessions_student_course
    ON course.tutor_sessions (student_id, course_id, last_active DESC);

CREATE TABLE IF NOT EXISTS course.tutor_messages (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id    UUID NOT NULL REFERENCES course.tutor_sessions (id) ON DELETE CASCADE,
    role          TEXT NOT NULL CHECK (role IN ('user', 'assistant', 'system')),
    content       TEXT NOT NULL,
    citations     JSONB,
    concept_tags  UUID[],
    token_count   INT,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE course.tutor_messages IS
    'Individual turns within a tutor session, with optional citations and concept tags (plan 19.1).';

CREATE INDEX IF NOT EXISTS idx_tutor_messages_session_created
    ON course.tutor_messages (session_id, created_at);

ALTER TABLE settings.platform_app_settings
    ADD COLUMN IF NOT EXISTS ff_persistent_tutor BOOLEAN;

COMMENT ON COLUMN settings.platform_app_settings.ff_persistent_tutor IS
    'Enables persistent named AI tutor sessions with RAG citations (plan 19.1).';
