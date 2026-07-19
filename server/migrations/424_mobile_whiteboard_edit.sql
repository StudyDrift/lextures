-- MOB.6: staged rollout for course whiteboard authoring on iOS and Android.
ALTER TABLE settings.platform_app_settings
    ADD COLUMN IF NOT EXISTS ff_mobile_whiteboard_edit BOOLEAN;

COMMENT ON COLUMN settings.platform_app_settings.ff_mobile_whiteboard_edit IS
    'MOB.6: Course whiteboard create/edit/delete on iOS and Android. Default OFF.';
