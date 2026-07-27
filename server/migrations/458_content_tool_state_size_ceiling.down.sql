-- CT.17 down: restore historic 64 KiB state_json CHECK.

ALTER TABLE course.content_tool_states
  DROP CONSTRAINT IF EXISTS content_tool_states_size;

ALTER TABLE course.content_tool_states
  ADD CONSTRAINT content_tool_states_size
  CHECK (pg_column_size(state_json) <= 65536);
