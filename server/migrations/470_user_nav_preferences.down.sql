ALTER TABLE settings.platform_app_settings
    DROP COLUMN IF EXISTS ff_navigation_v2;

DROP TABLE IF EXISTS settings.user_nav_preferences;
