-- Dedicated model for AI-generated vibe activities.
ALTER TABLE "user".user_ai_settings
    ADD COLUMN IF NOT EXISTS vibe_activity_model_id TEXT NOT NULL
        DEFAULT 'arcee-ai/trinity-mini:free';
