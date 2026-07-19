-- MOB.5: staged rollout for interactive live quizzes on iOS and Android.
ALTER TABLE settings.platform_app_settings
    ADD COLUMN IF NOT EXISTS ff_mobile_live_quiz BOOLEAN;

COMMENT ON COLUMN settings.platform_app_settings.ff_mobile_live_quiz IS
    'MOB.5: Interactive live quizzes (join/play) on iOS and Android. Default OFF.';
