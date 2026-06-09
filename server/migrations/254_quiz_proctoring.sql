-- Proctoring integration (plan 14.9): per-quiz LTI proctoring config and session tracking.

CREATE TABLE course.quiz_proctoring_config (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    structure_item_id UUID NOT NULL UNIQUE REFERENCES course.course_structure_items (id) ON DELETE CASCADE,
    external_tool_id UUID NOT NULL REFERENCES settings.lti_external_tools (id) ON DELETE RESTRICT,
    vendor           TEXT NOT NULL CHECK (vendor IN ('honorlock', 'respondus', 'proctu', 'examity')),
    required         BOOLEAN NOT NULL DEFAULT FALSE,
    settings         JSONB NOT NULL DEFAULT '{}',
    created_by       UUID REFERENCES "user".users (id) ON DELETE SET NULL,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE course.quiz_proctoring_sessions (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    attempt_id       UUID NOT NULL REFERENCES course.quiz_attempts (id) ON DELETE CASCADE,
    vendor           TEXT NOT NULL,
    vendor_session_id TEXT,
    status           TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'active', 'complete', 'flagged')),
    flag_count       INT NOT NULL DEFAULT 0,
    review_url       TEXT,
    started_at       TIMESTAMPTZ,
    completed_at     TIMESTAMPTZ,
    raw_callback     JSONB,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_quiz_proctoring_sessions_attempt ON course.quiz_proctoring_sessions (attempt_id);

-- Feature flag column for the proctoring integration (plan 14.9).
ALTER TABLE settings.platform_app_settings
    ADD COLUMN IF NOT EXISTS ff_proctoring_integration BOOLEAN;
