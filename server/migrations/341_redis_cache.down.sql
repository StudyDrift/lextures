-- Rollback for 341_redis_cache.sql
ALTER TABLE settings.platform_app_settings DROP COLUMN IF EXISTS ff_redis_cache;
