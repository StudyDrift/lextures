-- M11.5 / MOB.1: expose mobile create-course entry flag via platform settings (was client-decode only).
ALTER TABLE settings.platform_app_settings
    ADD COLUMN IF NOT EXISTS ff_mobile_create_course BOOLEAN;

COMMENT ON COLUMN settings.platform_app_settings.ff_mobile_create_course IS
    'M11.5: Mobile New course entry + basic create wizard. Default OFF.';
