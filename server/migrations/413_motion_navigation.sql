-- AN.2: kill-switch for launch & navigation transitions.
ALTER TABLE settings.platform_app_settings
    ADD COLUMN IF NOT EXISTS ff_motion_navigation BOOLEAN;

COMMENT ON COLUMN settings.platform_app_settings.ff_motion_navigation IS
    'AN.2: App launch & navigation transitions (splash handoff, route/screen/tab motion). Default ON; set false to disable instantly.';
