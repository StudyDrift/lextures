-- Migration 203: Per-user timezone and course-level timezone (plan 11.4)

ALTER TABLE "user".users
    ADD COLUMN IF NOT EXISTS timezone TEXT DEFAULT NULL;

COMMENT ON COLUMN "user".users.timezone IS 'IANA timezone identifier for deadline display (e.g. America/New_York). NULL falls back to course timezone or UTC.';

ALTER TABLE course.courses
    ADD COLUMN IF NOT EXISTS course_timezone TEXT DEFAULT NULL;

COMMENT ON COLUMN course.courses.course_timezone IS 'Instructor-authoritative IANA timezone for deadline intent; shown alongside student-localized times.';
