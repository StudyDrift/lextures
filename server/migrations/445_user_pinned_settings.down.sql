-- Down: Plan PS.2 pinned settings table and platform flag.

DROP TABLE IF EXISTS settings.user_pinned_settings;

ALTER TABLE settings.platform_app_settings
    DROP COLUMN IF EXISTS ff_pinned_settings;
