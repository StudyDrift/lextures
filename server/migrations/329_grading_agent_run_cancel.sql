-- GA-M6 — Cancel / stop a running grading-agent batch.

ALTER TABLE assessment.grading_agent_runs
    ADD COLUMN IF NOT EXISTS cancelled_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS cancelled_by UUID REFERENCES "user".users (id);

COMMENT ON COLUMN assessment.grading_agent_runs.cancelled_at IS
    'When the run was cancelled by an instructor (GA-M6).';
COMMENT ON COLUMN assessment.grading_agent_runs.cancelled_by IS
    'User who cancelled the run (GA-M6).';

ALTER TABLE settings.platform_app_settings
    ADD COLUMN IF NOT EXISTS grader_agent_cancel_run_enabled BOOLEAN;

COMMENT ON COLUMN settings.platform_app_settings.grader_agent_cancel_run_enabled IS
    'When true, instructors may cancel in-progress grading-agent batch runs (GA-M6).';
