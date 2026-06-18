-- Dedicated model for the instructor-authored grading agent (SpeedGrader dry-run and batch runs).
ALTER TABLE "user".user_ai_settings
    ADD COLUMN IF NOT EXISTS grader_agent_model_id TEXT NOT NULL
        DEFAULT 'arcee-ai/trinity-mini:free';