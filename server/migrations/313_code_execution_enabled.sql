-- 313: Code execution capability for grading agent Code Test Runner node (plan 19.17.7).

ALTER TABLE settings.platform_app_settings
    ADD COLUMN IF NOT EXISTS code_execution_enabled BOOLEAN;

COMMENT ON COLUMN settings.platform_app_settings.code_execution_enabled IS
    'When true, enables sandboxed code execution (quiz code questions and grader agent Code Test Runner node).';
