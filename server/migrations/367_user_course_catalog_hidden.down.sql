-- Companion to: 367_user_course_catalog_hidden.sql

DELETE FROM course.user_course_catalog_hidden;

DROP INDEX IF EXISTS course.idx_user_course_catalog_hidden_user;

DROP TABLE IF EXISTS course.user_course_catalog_hidden;