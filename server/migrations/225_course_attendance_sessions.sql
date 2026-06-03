-- Course Attendance sessions (roll call / self report) with optional gradebook column.

ALTER TABLE course.courses
    ADD COLUMN IF NOT EXISTS attendance_enabled boolean NOT NULL DEFAULT false;

COMMENT ON COLUMN course.courses.attendance_enabled IS
    'When true, the Attendance tool is available in this course (nav + API).';

CREATE TABLE IF NOT EXISTS course.attendance_sessions (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    course_id           UUID NOT NULL REFERENCES course.courses (id) ON DELETE CASCADE,
    section_id          UUID REFERENCES course.course_sections (id) ON DELETE SET NULL,
    structure_item_id   UUID REFERENCES course.course_structure_items (id) ON DELETE SET NULL,
    title               TEXT NOT NULL,
    collection_method   TEXT NOT NULL CHECK (collection_method IN ('roll_call', 'self_report')),
    session_date        DATE NOT NULL,
    opens_at            TIMESTAMPTZ,
    closes_at           TIMESTAMPTZ,
    status              TEXT NOT NULL DEFAULT 'open'
                            CHECK (status IN ('open', 'closed')),
    gradebook_enabled   BOOLEAN NOT NULL DEFAULT false,
    points_possible     INTEGER CHECK (points_possible IS NULL OR points_possible > 0),
    tardy_points_ratio  NUMERIC(3,2) NOT NULL DEFAULT 0.5
                            CHECK (tardy_points_ratio >= 0 AND tardy_points_ratio <= 1),
    created_by          UUID NOT NULL REFERENCES "user".users (id),
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    closed_at           TIMESTAMPTZ,
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_attendance_sessions_course_date
    ON course.attendance_sessions (course_id, session_date DESC);

CREATE TABLE IF NOT EXISTS course.attendance_session_records (
    session_id      UUID NOT NULL REFERENCES course.attendance_sessions (id) ON DELETE CASCADE,
    student_user_id UUID NOT NULL REFERENCES "user".users (id) ON DELETE CASCADE,
    status          TEXT NOT NULL DEFAULT 'not_recorded'
                        CHECK (status IN ('present', 'absent', 'tardy', 'excused', 'not_recorded')),
    source          TEXT NOT NULL DEFAULT 'instructor'
                        CHECK (source IN ('instructor', 'self', 'override')),
    recorded_by     UUID REFERENCES "user".users (id),
    recorded_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (session_id, student_user_id)
);

CREATE INDEX IF NOT EXISTS idx_attendance_session_records_student
    ON course.attendance_session_records (student_user_id);

ALTER TABLE course.course_structure_items DROP CONSTRAINT IF EXISTS course_structure_items_kind_check;
ALTER TABLE course.course_structure_items
    ADD CONSTRAINT course_structure_items_kind_check
    CHECK (kind IN (
        'module', 'heading', 'content_page', 'assignment', 'quiz',
        'external_link', 'survey', 'lti_link', 'h5p', 'vibe_activity', 'attendance'
    ));

ALTER TABLE course.course_structure_items DROP CONSTRAINT IF EXISTS course_structure_items_parent_child_kind_check;
ALTER TABLE course.course_structure_items
    ADD CONSTRAINT course_structure_items_parent_child_kind_check
    CHECK (
        parent_id IS NULL
        OR kind IN (
            'heading', 'content_page', 'assignment', 'quiz',
            'external_link', 'survey', 'lti_link', 'h5p', 'vibe_activity', 'attendance'
        )
    );
