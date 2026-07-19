-- MOB.8: staged rollout for board templates/export/present/governance on iOS/Android.
ALTER TABLE settings.platform_app_settings
    ADD COLUMN IF NOT EXISTS ff_mobile_boards_advanced BOOLEAN;

COMMENT ON COLUMN settings.platform_app_settings.ff_mobile_boards_advanced IS
    'MOB.8: Board templates, duplication, export, present mode, and admin governance on mobile. Default OFF.';
