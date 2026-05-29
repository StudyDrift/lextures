-- Plan 12.7: High-contrast and reduced-motion columns on user_reading_preferences.
-- The table was created by migration 211 (plan 12.6); this migration extends it.

ALTER TABLE settings.user_reading_preferences
    ADD COLUMN IF NOT EXISTS high_contrast BOOLEAN NOT NULL DEFAULT false,
    ADD COLUMN IF NOT EXISTS reduce_motion BOOLEAN NOT NULL DEFAULT false;

COMMENT ON COLUMN settings.user_reading_preferences.high_contrast IS
    'Plan 12.7: When true, activates the 7:1-contrast CSS layer on every page the user visits.';
COMMENT ON COLUMN settings.user_reading_preferences.reduce_motion IS
    'Plan 12.7: When true, suppresses all CSS transitions and keyframe animations platform-wide.';

ALTER TABLE settings.platform_app_settings
    ADD COLUMN IF NOT EXISTS ff_high_contrast_reduced_motion BOOLEAN;

COMMENT ON COLUMN settings.platform_app_settings.ff_high_contrast_reduced_motion IS
    'Plan 12.7: Enables the high-contrast / reduced-motion preference panel and API endpoints.';
