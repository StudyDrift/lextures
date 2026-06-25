-- GA-M1 — Persistent review queue & run history indexes.

CREATE INDEX IF NOT EXISTS idx_grading_agent_runs_config_created
    ON assessment.grading_agent_runs (config_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_grading_agent_results_config_status_live
    ON assessment.grading_agent_results (config_id, status)
    WHERE is_dry_run = false;

ALTER TABLE assessment.grading_agent_results
    ADD COLUMN IF NOT EXISTS resolved_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS resolved_by UUID REFERENCES "user".users (id);

COMMENT ON COLUMN assessment.grading_agent_results.resolved_at IS
    'When a held or flagged result was cleared by staff (GA-M1).';
COMMENT ON COLUMN assessment.grading_agent_results.resolved_by IS
    'Staff user who cleared a held or flagged result (GA-M1).';

ALTER TABLE settings.platform_app_settings
    ADD COLUMN IF NOT EXISTS grader_agent_review_inbox_enabled BOOLEAN;

COMMENT ON COLUMN settings.platform_app_settings.grader_agent_review_inbox_enabled IS
    'Enables the persistent grading-agent review inbox and run history (GA-M1).';