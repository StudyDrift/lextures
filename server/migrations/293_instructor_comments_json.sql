-- Structured SpeedGrader comment thread (Canvas import + in-app feedback).
ALTER TABLE course.course_grades
    ADD COLUMN IF NOT EXISTS instructor_comments_json JSONB;

COMMENT ON COLUMN course.course_grades.instructor_comments_json IS
    'Ordered JSON array of grade feedback comments with author, body, and timestamps.';