-- Track last course access per enrollment for roster "Last access" (issue #644).

ALTER TABLE course.course_enrollments
    ADD COLUMN IF NOT EXISTS last_course_access_at TIMESTAMPTZ;

COMMENT ON COLUMN course.course_enrollments.last_course_access_at IS
    'Most recent time this enrollee accessed the course (API activity). Used by People/Enrollments roster.';

CREATE INDEX IF NOT EXISTS idx_course_enrollments_last_access
    ON course.course_enrollments (course_id, last_course_access_at DESC NULLS LAST)
    WHERE active = true;
