-- Plan 13.11: age-appropriate UI mode override stored on user_reading_preferences.
-- Valid values: NULL (derive from grade_level) | 'k2' | 'elementary' | 'standard'.
ALTER TABLE settings.user_reading_preferences
    ADD COLUMN IF NOT EXISTS ui_mode_override TEXT
    CONSTRAINT user_reading_preferences_ui_mode_override_check
    CHECK (ui_mode_override IS NULL OR ui_mode_override IN ('k2', 'elementary', 'standard'));
