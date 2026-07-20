-- AN.7: kill-switch for delight & progress moments (progress fills, quiz feedback, achievements).
ALTER TABLE settings.platform_app_settings
    ADD COLUMN IF NOT EXISTS ff_motion_delight BOOLEAN;

COMMENT ON COLUMN settings.platform_app_settings.ff_motion_delight IS
    'AN.7: Delight & progress moments (animated progress, quiz answer feedback, achievement bursts, leaderboard count-up). Default ON; set false to disable instantly. Collapsed into ff_motion_navigation.';
