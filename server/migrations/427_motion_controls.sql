-- AN.6: kill-switch for control micro-interactions (press, toggle, tabs, validation, haptics).
ALTER TABLE settings.platform_app_settings
    ADD COLUMN IF NOT EXISTS ff_motion_controls BOOLEAN;

COMMENT ON COLUMN settings.platform_app_settings.ff_motion_controls IS
    'AN.6: Control micro-interactions (press scale, toggle/checkbox/tab indicators, validation shake, loading buttons, haptics). Default ON; set false to disable instantly. Collapsed into ff_motion_navigation.';
