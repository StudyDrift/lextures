-- GA-M3 — Suggest-only batch run mode + suggest-mode feature flag.

ALTER TABLE assessment.grading_agent_runs
    ADD COLUMN IF NOT EXISTS mode TEXT NOT NULL DEFAULT 'apply'
    CHECK (mode IN ('suggest', 'apply'));

COMMENT ON COLUMN assessment.grading_agent_runs.mode IS
    'Whether the run writes grades immediately (apply) or holds suggestions for review (suggest) (GA-M3).';

UPDATE assessment.grading_agent_runs SET mode = 'apply' WHERE mode IS NULL;

ALTER TABLE settings.platform_app_settings
    ADD COLUMN IF NOT EXISTS grader_agent_suggest_mode_enabled BOOLEAN;

COMMENT ON COLUMN settings.platform_app_settings.grader_agent_suggest_mode_enabled IS
    'Enables suggest-only vs auto-apply run mode and bulk review actions (GA-M3).';