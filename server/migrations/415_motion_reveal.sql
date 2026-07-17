-- AN.3: kill-switch for skeleton→content load choreography.
ALTER TABLE settings.platform_app_settings
    ADD COLUMN IF NOT EXISTS ff_motion_reveal BOOLEAN;

COMMENT ON COLUMN settings.platform_app_settings.ff_motion_reveal IS
    'AN.3: Load choreography (skeleton→content crossfade, staggered reveal). Default ON; set false to disable instantly.';
