-- 311_grading_agent_workflow.sql
-- Plan 19.17 — store the visual workflow graph for the grading agent.

ALTER TABLE assessment.grading_agent_configs
    ADD COLUMN IF NOT EXISTS workflow_graph JSONB;

COMMENT ON COLUMN assessment.grading_agent_configs.workflow_graph IS
    'React Flow node graph authored in the SpeedGrader workflow canvas (plan 19.17). '
    'NULL for legacy prompt-only configs, which are synthesized into a default graph on read.';

ALTER TABLE assessment.grading_agent_runs
    ADD COLUMN IF NOT EXISTS authored_via TEXT;

COMMENT ON COLUMN assessment.grading_agent_runs.authored_via IS
    'Authoring surface for the run: canvas, form, or NULL for legacy prompt-only configs.';
