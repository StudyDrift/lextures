-- Plan 12.6: Dyslexia-friendly font, line spacing, and reading ruler.

CREATE TABLE IF NOT EXISTS settings.user_reading_preferences (
    user_id        UUID PRIMARY KEY REFERENCES "user".users(id) ON DELETE CASCADE,
    font_face      TEXT NOT NULL DEFAULT 'default',
    letter_spacing TEXT NOT NULL DEFAULT 'normal',
    word_spacing   TEXT NOT NULL DEFAULT 'normal',
    line_height    TEXT NOT NULL DEFAULT 'normal',
    ruler_enabled  BOOLEAN NOT NULL DEFAULT false,
    ruler_color    TEXT NOT NULL DEFAULT 'yellow',
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT urp_font_face_check      CHECK (font_face      IN ('default', 'open-dyslexic', 'atkinson', 'system')),
    CONSTRAINT urp_letter_spacing_check CHECK (letter_spacing IN ('normal', 'wide', 'wider')),
    CONSTRAINT urp_word_spacing_check   CHECK (word_spacing   IN ('normal', 'wide', 'wider')),
    CONSTRAINT urp_line_height_check    CHECK (line_height    IN ('normal', 'tall', 'taller')),
    CONSTRAINT urp_ruler_color_check    CHECK (ruler_color    IN ('yellow', 'grey'))
);

COMMENT ON TABLE settings.user_reading_preferences IS
    'Per-user dyslexia-friendly reading preferences: font, spacing, and reading ruler (plan 12.6).';

ALTER TABLE settings.platform_app_settings
    ADD COLUMN IF NOT EXISTS ff_reading_preferences BOOLEAN NOT NULL DEFAULT FALSE;

COMMENT ON COLUMN settings.platform_app_settings.ff_reading_preferences IS
    'Plan 12.6 (ff_reading_preferences): enables the Reading Preferences panel, font selector, spacing controls, and reading ruler UI. Default false; flip true after QA sign-off.';
