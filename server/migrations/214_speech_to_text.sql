-- Plan 12.9: Speech-to-text input for student responses (WCAG SC 2.1.1, 4.1.2, 4.1.3).

ALTER TABLE settings.platform_app_settings
    ADD COLUMN IF NOT EXISTS speech_to_text_enabled BOOLEAN;

COMMENT ON COLUMN settings.platform_app_settings.speech_to_text_enabled IS
    'Plan 12.9: Enables speech-to-text dictation in the block editor and quiz short-answer fields.';

CREATE TABLE IF NOT EXISTS settings.user_reading_preferences (
    user_id UUID PRIMARY KEY REFERENCES "user".users (id) ON DELETE CASCADE,
    stt_enabled BOOLEAN NOT NULL DEFAULT false,
    stt_language TEXT NOT NULL DEFAULT 'en-US',
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

COMMENT ON TABLE settings.user_reading_preferences IS
    'Per-user reading and accessibility display preferences (extended by plans 12.6–12.8).';
COMMENT ON COLUMN settings.user_reading_preferences.stt_enabled IS
    'Plan 12.9: User opt-in for speech-to-text dictation controls.';
COMMENT ON COLUMN settings.user_reading_preferences.stt_language IS
    'Plan 12.9: BCP 47 language tag passed to SpeechRecognition.lang.';

ALTER TABLE course.student_accommodations
    ADD COLUMN IF NOT EXISTS speech_to_text_enabled BOOLEAN NOT NULL DEFAULT false;

COMMENT ON COLUMN course.student_accommodations.speech_to_text_enabled IS
    'Plan 12.9: Auto-enables speech-to-text dictation for this learner (IEP/504 accommodation).';
