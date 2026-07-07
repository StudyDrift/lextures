-- Companion to: 357_ff_parent_portal_v2.sql

ALTER TABLE settings.platform_app_settings
    DROP COLUMN IF EXISTS ff_parent_portal_v2;
