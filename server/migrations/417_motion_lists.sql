-- AN.4: kill-switch for list / collection mutation motion.
ALTER TABLE settings.platform_app_settings
    ADD COLUMN IF NOT EXISTS ff_motion_lists BOOLEAN;

COMMENT ON COLUMN settings.platform_app_settings.ff_motion_lists IS
    'AN.4: List/collection motion (insert/remove/reorder, drag lift). Default ON; set false to disable instantly.';
