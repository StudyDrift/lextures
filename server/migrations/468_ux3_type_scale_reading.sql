-- UX.3 — Typography and reading system: text scale preference + base-size feature flag.

ALTER TABLE settings.user_reading_preferences
    ADD COLUMN IF NOT EXISTS text_scale NUMERIC(4, 3) NOT NULL DEFAULT 1.0;

ALTER TABLE settings.user_reading_preferences
    DROP CONSTRAINT IF EXISTS urp_text_scale_check;

ALTER TABLE settings.user_reading_preferences
    ADD CONSTRAINT urp_text_scale_check
    CHECK (text_scale IN (1.0, 1.125, 1.25, 1.5));

COMMENT ON COLUMN settings.user_reading_preferences.text_scale IS
    'UX.3: multiplies the semantic type scale (1.0 | 1.125 | 1.25 | 1.5). Applied via --lx-type-scale.';

ALTER TABLE settings.platform_app_settings
    ADD COLUMN IF NOT EXISTS ff_type_scale BOOLEAN NOT NULL DEFAULT FALSE;

COMMENT ON COLUMN settings.platform_app_settings.ff_type_scale IS
    'UX.3 (ff_type_scale): raises default body size to 16px. Roles/lint ship unflagged; this gates the base-size raise only. Default false until dogfood.';
