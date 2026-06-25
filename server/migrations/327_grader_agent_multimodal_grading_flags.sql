-- Platform flags for text-entry and vision grading paths (GA-M2).

ALTER TABLE settings.platform_app_settings
    ADD COLUMN IF NOT EXISTS grader_agent_text_entry_grading_enabled BOOLEAN,
    ADD COLUMN IF NOT EXISTS grader_agent_vision_grading_enabled BOOLEAN;

COMMENT ON COLUMN settings.platform_app_settings.grader_agent_text_entry_grading_enabled IS
    'When true, the grading agent grades typed online text-entry submissions (GA-M2).';
COMMENT ON COLUMN settings.platform_app_settings.grader_agent_vision_grading_enabled IS
    'When true, the grading agent may use vision models for image-only or scanned submissions (GA-M2).';