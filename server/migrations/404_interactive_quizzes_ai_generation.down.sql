-- IQ.10 down — AI-assisted quiz kit generation.

DELETE FROM settings.system_prompts WHERE key = 'live_quiz_kit_generation';

ALTER TABLE settings.platform_app_settings
    DROP COLUMN IF EXISTS ff_iq_ai_generation;

ALTER TABLE quizgame.questions
    DROP CONSTRAINT IF EXISTS quizgame_questions_source_chk;

ALTER TABLE quizgame.questions
    DROP COLUMN IF EXISTS generation_confidence,
    DROP COLUMN IF EXISTS generation_job_id,
    DROP COLUMN IF EXISTS needs_review,
    DROP COLUMN IF EXISTS source;

DROP TABLE IF EXISTS quizgame.generation_jobs;
