-- CT.17: raise content_tool_states size CHECK to platform ceiling (256 KiB)
-- so tools may declare storage.maxStateBytes above the historic 64 KiB default
-- (code_sandbox uses 128000).
-- See docs/runbooks/database-migration-rollback.md

ALTER TABLE course.content_tool_states
  DROP CONSTRAINT IF EXISTS content_tool_states_size;

ALTER TABLE course.content_tool_states
  ADD CONSTRAINT content_tool_states_size
  CHECK (pg_column_size(state_json) <= 262144);

COMMENT ON CONSTRAINT content_tool_states_size ON course.content_tool_states IS
  'CT.17: Platform ceiling matches contenttools.PlatformMaxStateBytes (256 KiB).';
