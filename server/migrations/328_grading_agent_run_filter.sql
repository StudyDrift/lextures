-- GA-M5 — Section / group / student-scoped grading agent runs.

ALTER TABLE assessment.grading_agent_runs
    ADD COLUMN IF NOT EXISTS filter JSONB;

COMMENT ON COLUMN assessment.grading_agent_runs.filter IS
    'Optional run target filter: sectionId, groupId, and/or submissionIds (GA-M5).';

ALTER TABLE settings.platform_app_settings
    ADD COLUMN IF NOT EXISTS grader_agent_run_filters_enabled BOOLEAN;

COMMENT ON COLUMN settings.platform_app_settings.grader_agent_run_filters_enabled IS
    'When true, grading agent runs may target a section, group, or explicit submission selection (GA-M5).';
