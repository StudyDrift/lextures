-- Companion to: 463_course_checklist_review_markers_2.sql

ALTER TABLE course.courses
    DROP COLUMN IF EXISTS accommodations_reviewed_at,
    DROP COLUMN IF EXISTS integrity_settings_reviewed_at;
