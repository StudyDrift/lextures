DROP TABLE IF EXISTS tenant.org_mobile_link_handling;

ALTER TABLE settings.platform_app_settings
    DROP CONSTRAINT IF EXISTS platform_app_settings_mobile_link_handling_check;

ALTER TABLE settings.platform_app_settings
    DROP COLUMN IF EXISTS mobile_link_handling;

ALTER TABLE settings.platform_app_settings
    DROP COLUMN IF EXISTS ff_mobile_in_app_browser;
