-- MOB.4: staged rollout for adding course enrollments from iOS and Android.
ALTER TABLE settings.platform_app_settings
    ADD COLUMN IF NOT EXISTS ff_mobile_enrollment_add BOOLEAN;

COMMENT ON COLUMN settings.platform_app_settings.ff_mobile_enrollment_add IS
    'MOB.4: Add course enrollments from iOS and Android People roster. Default OFF.';
