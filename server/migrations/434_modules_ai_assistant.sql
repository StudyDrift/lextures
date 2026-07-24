-- Modules AI assistant: instructor chat pane on the Modules page (course feature flag).

ALTER TABLE course.courses
  ADD COLUMN IF NOT EXISTS modules_ai_assistant_enabled BOOLEAN NOT NULL DEFAULT FALSE;

COMMENT ON COLUMN course.courses.modules_ai_assistant_enabled IS
  'When true, course staff can open the Modules AI assistant chat pane (requires platform AI configured).';
