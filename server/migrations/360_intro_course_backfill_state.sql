-- IC02: tracks one-time intro course enrollment backfill progress (resumable cursor).

CREATE TABLE IF NOT EXISTS settings.intro_course_backfill (
    id             BOOLEAN PRIMARY KEY DEFAULT TRUE CHECK (id),
    started_at     TIMESTAMPTZ,
    completed_at   TIMESTAMPTZ,
    last_user_id   UUID,
    enrolled_count BIGINT NOT NULL DEFAULT 0,
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

COMMENT ON TABLE settings.intro_course_backfill IS
    'Singleton row tracking the one-time intro course student enrollment backfill (IC02).';

INSERT INTO settings.intro_course_backfill (id)
VALUES (TRUE)
ON CONFLICT (id) DO NOTHING;