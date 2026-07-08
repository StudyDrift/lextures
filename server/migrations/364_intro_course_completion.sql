-- IC05: per-student intro course completion records (progress is derived; this stores the finish line).

CREATE TABLE IF NOT EXISTS settings.intro_course_completions (
    user_id       UUID PRIMARY KEY REFERENCES "user".users (id) ON DELETE CASCADE,
    completed_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    final_grade   DOUBLE PRECISION,
    credential_id UUID REFERENCES credentials.issued_credentials (id) ON DELETE SET NULL,
    event_sent    BOOLEAN NOT NULL DEFAULT FALSE,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

COMMENT ON TABLE settings.intro_course_completions IS
    'Intro course completion snapshot per learner (IC05). Progress is derived from grades; this row is set once.';

CREATE INDEX IF NOT EXISTS idx_intro_course_completions_completed_at
    ON settings.intro_course_completions (completed_at DESC);