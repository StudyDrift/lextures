-- CT.9 down — Content Tools marketplace.

ALTER TABLE settings.platform_app_settings
    DROP COLUMN IF EXISTS ff_content_tool_marketplace;

DROP TABLE IF EXISTS toolmarket.tool_lifecycle_events;
DROP TABLE IF EXISTS toolmarket.tool_access_grants;
DROP TABLE IF EXISTS toolmarket.tool_installations;
DROP TABLE IF EXISTS toolmarket.tool_releases;
DROP TABLE IF EXISTS toolmarket.tools;
DROP SCHEMA IF EXISTS toolmarket;
