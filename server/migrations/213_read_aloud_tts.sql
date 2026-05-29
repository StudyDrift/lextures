-- Plan 12.8: Text-to-speech read-aloud (WCAG SC 1.4.1, 2.1.1).
-- Extends settings.user_reading_preferences (plan 12.6) with TTS columns.
-- read_aloud_enabled / ff_read_aloud: platform feature flags (Settings → Global platform).
-- student_accommodations.tts_enabled: IEP read-aloud accommodation (partial 12.10).

ALTER TABLE settings.user_reading_preferences
    ADD COLUMN IF NOT EXISTS tts_enabled BOOLEAN NOT NULL DEFAULT false,
    ADD COLUMN IF NOT EXISTS tts_speed NUMERIC(3, 2) NOT NULL DEFAULT 1.0,
    ADD COLUMN IF NOT EXISTS tts_voice_name TEXT;

ALTER TABLE settings.user_reading_preferences
    DROP CONSTRAINT IF EXISTS urp_tts_speed_check;

ALTER TABLE settings.user_reading_preferences
    ADD CONSTRAINT urp_tts_speed_check CHECK (tts_speed >= 0.75 AND tts_speed <= 2.0);

COMMENT ON COLUMN settings.user_reading_preferences.tts_enabled IS
    'When true, read-aloud is the user default on content pages (plan 12.8).';
COMMENT ON COLUMN settings.user_reading_preferences.tts_speed IS
    'Playback rate multiplier (0.75–2.0) (plan 12.8).';
COMMENT ON COLUMN settings.user_reading_preferences.tts_voice_name IS
    'Browser SpeechSynthesis voice name; NULL = system default (plan 12.8).';

ALTER TABLE settings.platform_app_settings
    ADD COLUMN IF NOT EXISTS read_aloud_enabled BOOLEAN,
    ADD COLUMN IF NOT EXISTS ff_read_aloud BOOLEAN;

COMMENT ON COLUMN settings.platform_app_settings.read_aloud_enabled IS
    'Plan 12.8: Master toggle for in-context read-aloud on course content pages.';
COMMENT ON COLUMN settings.platform_app_settings.ff_read_aloud IS
    'Plan 12.8: When true (and read-aloud enabled), exposes read-aloud controls to learners.';

ALTER TABLE course.student_accommodations
    ADD COLUMN IF NOT EXISTS tts_enabled BOOLEAN NOT NULL DEFAULT false;

COMMENT ON COLUMN course.student_accommodations.tts_enabled IS
    'Plan 12.8: Auto-enable read-aloud for this learner (IEP/504 accommodation).';
