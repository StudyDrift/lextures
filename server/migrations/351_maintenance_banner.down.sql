DROP TABLE IF EXISTS platform.banners;
DROP TYPE IF EXISTS platform.banner_scope;
DROP TYPE IF EXISTS platform.banner_severity;
ALTER TABLE settings.platform_app_settings DROP COLUMN IF EXISTS maintenance_banner_enabled;