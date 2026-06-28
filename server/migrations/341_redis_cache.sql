-- Plan 17.5: Redis object cache feature flag (default off; rollout via Settings → Global platform).

ALTER TABLE settings.platform_app_settings
    ADD COLUMN IF NOT EXISTS ff_redis_cache BOOLEAN;

COMMENT ON COLUMN settings.platform_app_settings.ff_redis_cache IS
    'When true, hot-path reads (course structure, enrollments, public catalog, calendar feeds) use the shared Redis object cache.';
