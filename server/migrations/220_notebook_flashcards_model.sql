-- Add a dedicated model column for AI-generated notebook flashcards.
-- Falls back to course_setup_model_id when not set.
ALTER TABLE "user".user_ai_settings
    ADD COLUMN IF NOT EXISTS notebook_flashcards_model_id TEXT NOT NULL
        DEFAULT 'arcee-ai/trinity-mini:free';
