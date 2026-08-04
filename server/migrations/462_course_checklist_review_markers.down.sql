ALTER TABLE course.course_syllabus
    DROP COLUMN IF EXISTS acceptance_decided_at;

ALTER TABLE course.courses
    DROP COLUMN IF EXISTS features_reviewed_at;
