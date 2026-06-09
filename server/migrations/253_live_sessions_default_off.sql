-- New courses should not show Live Sessions until instructors opt in.
-- Existing rows keep their current live_sessions_enabled value.

ALTER TABLE course.courses ALTER COLUMN live_sessions_enabled SET DEFAULT FALSE;
