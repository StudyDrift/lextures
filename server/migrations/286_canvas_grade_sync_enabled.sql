-- Per-course toggle: when enabled, grades saved in Lextures can be pushed back to the linked Canvas course.
ALTER TABLE course.courses
  ADD COLUMN IF NOT EXISTS canvas_grade_sync_enabled BOOLEAN NOT NULL DEFAULT FALSE;