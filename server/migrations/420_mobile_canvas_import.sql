-- MOB.2: staged rollout for mobile Canvas course import (credentials → select → live progress).
ALTER TABLE settings.platform_app_settings
    ADD COLUMN IF NOT EXISTS ff_mobile_canvas_import BOOLEAN;

COMMENT ON COLUMN settings.platform_app_settings.ff_mobile_canvas_import IS
    'MOB.2: Mobile Canvas course import wizard (credentials, scope, live progress). Default OFF.';
