-- Revert UX.7 navigation preferences / ff_navigation_v2 (rollback of 470).

ALTER TABLE settings.platform_app_settings
    DROP COLUMN IF EXISTS ff_navigation_v2;

DROP TABLE IF EXISTS settings.user_nav_preferences;
