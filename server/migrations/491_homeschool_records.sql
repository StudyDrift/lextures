-- Issue #774 — homeschool attendance/hours log and year-end transcript/portfolio settings.

CREATE TABLE IF NOT EXISTS "user".homeschool_records_settings (
    parent_user_id              UUID PRIMARY KEY REFERENCES "user".users(id) ON DELETE CASCADE,
    school_name                 TEXT NOT NULL DEFAULT '',
    school_year_start_month     INT NOT NULL DEFAULT 8
        CHECK (school_year_start_month BETWEEN 1 AND 12),
    school_year_start_day       INT NOT NULL DEFAULT 1
        CHECK (school_year_start_day BETWEEN 1 AND 31),
    updated_at                  TIMESTAMPTZ NOT NULL DEFAULT now()
);

COMMENT ON TABLE "user".homeschool_records_settings IS
    'Issue #774: family school name and school-year start used on hours logs and transcripts.';

CREATE TABLE IF NOT EXISTS "user".homeschool_hours_entries (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    parent_user_id   UUID NOT NULL REFERENCES "user".users(id) ON DELETE CASCADE,
    student_user_id  UUID NOT NULL REFERENCES "user".users(id) ON DELETE CASCADE,
    entry_date       DATE NOT NULL,
    subject          TEXT NOT NULL DEFAULT '',
    course_id        UUID REFERENCES course.courses(id) ON DELETE SET NULL,
    hours            NUMERIC(5, 2) NOT NULL
        CHECK (hours >= 0 AND hours <= 24),
    notes            TEXT NOT NULL DEFAULT '',
    source           TEXT NOT NULL DEFAULT 'manual'
        CHECK (source IN ('manual', 'offline')),
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS homeschool_hours_entries_student_date
    ON "user".homeschool_hours_entries (student_user_id, entry_date DESC);

CREATE INDEX IF NOT EXISTS homeschool_hours_entries_parent
    ON "user".homeschool_hours_entries (parent_user_id, student_user_id);

COMMENT ON TABLE "user".homeschool_hours_entries IS
    'Issue #774: parent-entered offline/manual instructional hours by subject for a managed learner or linked student.';
