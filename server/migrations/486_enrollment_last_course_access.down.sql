DROP INDEX IF EXISTS course.idx_course_enrollments_last_access;

ALTER TABLE course.course_enrollments
    DROP COLUMN IF EXISTS last_course_access_at;
