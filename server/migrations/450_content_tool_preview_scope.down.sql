-- Companion to: 450_content_tool_preview_scope.sql

DROP INDEX IF EXISTS course.idx_cts_preview_created;
DROP INDEX IF EXISTS course.uq_cts_instance_enrollment_real;

-- Remove preview rows before restoring uniqueness (preview pairs may duplicate enrollment keys).
DELETE FROM course.content_tool_states WHERE scope = 'preview';

ALTER TABLE course.content_tool_states
    DROP CONSTRAINT IF EXISTS content_tool_states_scope_check;
ALTER TABLE course.content_tool_states
    DROP COLUMN IF EXISTS scope;

ALTER TABLE course.content_tool_states
    ADD CONSTRAINT content_tool_states_instance_id_enrollment_id_key UNIQUE (instance_id, enrollment_id);
