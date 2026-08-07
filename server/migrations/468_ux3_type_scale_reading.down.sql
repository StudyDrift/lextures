ALTER TABLE settings.user_reading_preferences
    DROP CONSTRAINT IF EXISTS urp_text_scale_check;

ALTER TABLE settings.user_reading_preferences
    DROP COLUMN IF EXISTS text_scale;

ALTER TABLE settings.platform_app_settings
    DROP COLUMN IF EXISTS ff_type_scale;
