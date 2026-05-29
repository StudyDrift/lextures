-- Plan 12.7: High-contrast and reduced-motion preferences (WCAG SC 1.4.3, 2.3.3).
-- Creates per-user reading preferences table and adds the feature flag.
--
-- user_reading_preferences stores opt-in accessibility overrides keyed by user_id.
-- ff_high_contrast_reduced_motion gates the API and panel UI (default off).

CREATE TABLE IF NOT EXISTS settings.user_reading_preferences (
    user_id     UUID        NOT NULL PRIMARY KEY REFERENCES "user".users(id) ON DELETE CASCADE,
    high_contrast  BOOLEAN NOT NULL DEFAULT false,
    reduce_motion  BOOLEAN NOT NULL DEFAULT false,
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE settings.user_reading_preferences IS
    'Plan 12.7: Per-user high-contrast and reduced-motion accessibility preferences.';
COMMENT ON COLUMN settings.user_reading_preferences.high_contrast IS
    'When true, activates the 7:1-contrast CSS layer on every page the user visits.';
COMMENT ON COLUMN settings.user_reading_preferences.reduce_motion IS
    'When true, suppresses all CSS transitions and keyframe animations platform-wide.';

ALTER TABLE settings.platform_app_settings
    ADD COLUMN IF NOT EXISTS ff_high_contrast_reduced_motion BOOLEAN;

COMMENT ON COLUMN settings.platform_app_settings.ff_high_contrast_reduced_motion IS
    'Plan 12.7: Enables the high-contrast / reduced-motion preference panel and API endpoints.';
