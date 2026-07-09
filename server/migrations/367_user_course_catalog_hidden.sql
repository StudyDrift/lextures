-- Per-user hidden courses on the Courses catalog (orthogonal to kanban placement).

CREATE TABLE IF NOT EXISTS course.user_course_catalog_hidden (
    user_id   UUID NOT NULL REFERENCES "user".users (id) ON DELETE CASCADE,
    course_id UUID NOT NULL REFERENCES course.courses (id) ON DELETE CASCADE,
    hidden_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, course_id)
);

CREATE INDEX IF NOT EXISTS idx_user_course_catalog_hidden_user
    ON course.user_course_catalog_hidden (user_id, hidden_at DESC);

COMMENT ON TABLE course.user_course_catalog_hidden IS
    'Courses the user has hidden from their Courses page catalog views.';

-- Backfill from legacy kanban Hidden column placements.
INSERT INTO course.user_course_catalog_hidden (user_id, course_id)
SELECT user_id, course_id
FROM course.user_course_kanban_placement
WHERE column_id = 'hidden'
ON CONFLICT DO NOTHING;

DELETE FROM course.user_course_kanban_placement
WHERE column_id = 'hidden';
