-- Companion to: 464_course_checklist_link_health.sql

DROP TABLE IF EXISTS course.course_checklist_link_health;

ALTER TABLE course.courses
    DROP COLUMN IF EXISTS a11y_reviewed_at,
    DROP COLUMN IF EXISTS student_preview_at,
    DROP COLUMN IF EXISTS last_export_at;
