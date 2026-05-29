-- Plan 12.9: Speech-to-text input for student responses (WCAG SC 2.1.1, 4.1.2, 4.1.3).
-- Extends plan 12.6 user_reading_preferences table (migration 211).

ALTER TABLE settings.platform_app_settings
    ADD COLUMN IF NOT EXISTS speech_to_text_enabled BOOLEAN;

COMMENT ON COLUMN settings.platform_app_settings.speech_to_text_enabled IS
    'Plan 12.9: Enables speech-to-text dictation in the block editor and quiz short-answer fields.';

ALTER TABLE settings.user_reading_preferences
    ADD COLUMN IF NOT EXISTS stt_enabled BOOLEAN NOT NULL DEFAULT false,
    ADD COLUMN IF NOT EXISTS stt_language TEXT NOT NULL DEFAULT 'en-US';

COMMENT ON COLUMN settings.user_reading_preferences.stt_enabled IS
    'Plan 12.9: User opt-in for speech-to-text dictation controls.';
COMMENT ON COLUMN settings.user_reading_preferences.stt_language IS
    'Plan 12.9: BCP 47 language tag passed to SpeechRecognition.lang.';

ALTER TABLE course.student_accommodations
    ADD COLUMN IF NOT EXISTS speech_to_text_enabled BOOLEAN NOT NULL DEFAULT false;

COMMENT ON COLUMN course.student_accommodations.speech_to_text_enabled IS
    'Plan 12.9: Auto-enables speech-to-text dictation for this learner (IEP/504 accommodation).';
