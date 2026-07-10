-- Companion to: 369_marketplace_course_provisioning.sql

DROP TABLE IF EXISTS settings.marketplace_course_items;
DROP TABLE IF EXISTS settings.marketplace_courses;

DROP INDEX IF EXISTS course.idx_courses_is_official;

ALTER TABLE course.courses
    DROP COLUMN IF EXISTS is_official;

DELETE FROM "user".users
WHERE id = 'a0000000-0000-4000-8000-000000000003'::uuid;
