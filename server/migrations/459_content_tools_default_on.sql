-- Content Tools is available by default on new courses (instructors can still disable).
-- Existing rows keep their current content_tools_enabled value.

ALTER TABLE course.courses ALTER COLUMN content_tools_enabled SET DEFAULT TRUE;
