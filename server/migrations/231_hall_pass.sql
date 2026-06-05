-- Plan 13.9: Hall pass + classroom-management signals (anonymous question queue).

CREATE SCHEMA IF NOT EXISTS classroom;

CREATE TABLE IF NOT EXISTS classroom.hall_passes (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    student_id      UUID        NOT NULL REFERENCES "user".users (id) ON DELETE CASCADE,
    section_id      UUID        NOT NULL REFERENCES course.course_sections (id) ON DELETE CASCADE,
    destination     TEXT        NOT NULL,
    estimated_mins  INTEGER     CHECK (estimated_mins IS NULL OR (estimated_mins > 0 AND estimated_mins <= 120)),
    status          TEXT        NOT NULL DEFAULT 'requested'
                                CHECK (status IN ('requested', 'approved', 'returned', 'denied')),
    requested_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    approved_at     TIMESTAMPTZ,
    returned_at     TIMESTAMPTZ,
    approved_by     UUID        REFERENCES "user".users (id) ON DELETE SET NULL
);

CREATE INDEX IF NOT EXISTS hall_passes_section_idx
    ON classroom.hall_passes (section_id, requested_at DESC);
CREATE INDEX IF NOT EXISTS hall_passes_student_idx
    ON classroom.hall_passes (student_id, requested_at DESC);
CREATE INDEX IF NOT EXISTS hall_passes_active_idx
    ON classroom.hall_passes (section_id)
    WHERE status IN ('requested', 'approved');

COMMENT ON TABLE classroom.hall_passes IS
    'Plan 13.9: Digital hall passes. FERPA: education record; only teacher, student, and school admin should view.';

CREATE TABLE IF NOT EXISTS classroom.anonymous_questions (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    course_id   UUID        NOT NULL REFERENCES course.courses (id) ON DELETE CASCADE,
    -- author_id is stored for moderation/abuse review by the teacher but never
    -- returned to other students. NULL when the author is no longer in the system.
    author_id   UUID        REFERENCES "user".users (id) ON DELETE SET NULL,
    question    TEXT        NOT NULL,
    addressed   BOOLEAN     NOT NULL DEFAULT FALSE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS anonymous_questions_course_idx
    ON classroom.anonymous_questions (course_id, created_at DESC);

COMMENT ON TABLE classroom.anonymous_questions IS
    'Plan 13.9: Anonymous question queue. Author is hidden from peers but visible to teacher for moderation.';

ALTER TABLE settings.platform_app_settings
    ADD COLUMN IF NOT EXISTS ff_classroom_signals BOOLEAN;

COMMENT ON COLUMN settings.platform_app_settings.ff_classroom_signals IS
    'Plan 13.9: Enables digital hall passes and anonymous question queue.';
