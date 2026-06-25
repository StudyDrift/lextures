-- Per-result input modality for grading-agent analytics (GA-M2).

ALTER TABLE assessment.grading_agent_results
    ADD COLUMN IF NOT EXISTS input_modality TEXT;

COMMENT ON COLUMN assessment.grading_agent_results.input_modality IS
    'How the submission was read: text, file, vision, or unreadable.';