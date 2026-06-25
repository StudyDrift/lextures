-- GA-M7 — Pre-run cost estimate, run cost aggregates, and optional per-run budget cap.

ALTER TABLE assessment.grading_agent_runs
    ADD COLUMN IF NOT EXISTS budget_usd NUMERIC(14, 8);

COMMENT ON COLUMN assessment.grading_agent_runs.budget_usd IS
    'Optional per-run AI spend cap in USD; remaining items are skipped when observed spend reaches this limit (GA-M7).';

ALTER TABLE settings.platform_app_settings
    ADD COLUMN IF NOT EXISTS grader_agent_cost_estimate_enabled BOOLEAN;

COMMENT ON COLUMN settings.platform_app_settings.grader_agent_cost_estimate_enabled IS
    'When true, the grading agent run popover shows submission count and approximate cost before starting a batch (GA-M7).';
