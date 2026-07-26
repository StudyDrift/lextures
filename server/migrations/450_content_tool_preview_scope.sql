-- CT.2 — Preview scope for instructor preview-as-student state (never pollutes real enrollment uniqueness).

ALTER TABLE course.content_tool_states
    ADD COLUMN IF NOT EXISTS scope TEXT NOT NULL DEFAULT 'enrollment'
        CHECK (scope IN ('enrollment', 'preview'));
COMMENT ON COLUMN course.content_tool_states.scope IS
    'enrollment = real learner work; preview = instructor preview-as-student, purged nightly (plan CT.2).';

-- Real state stays unique per (instance, enrollment); preview rows are excluded from that uniqueness.
-- Drop the UNIQUE constraint first (it owns the backing index); then recreate as a partial unique index.
ALTER TABLE course.content_tool_states
    DROP CONSTRAINT IF EXISTS content_tool_states_instance_id_enrollment_id_key;
DROP INDEX IF EXISTS course.content_tool_states_instance_id_enrollment_id_key;
CREATE UNIQUE INDEX IF NOT EXISTS uq_cts_instance_enrollment_real
    ON course.content_tool_states (instance_id, enrollment_id)
    WHERE scope = 'enrollment';
CREATE INDEX IF NOT EXISTS idx_cts_preview_created
    ON course.content_tool_states (created_at) WHERE scope = 'preview';
