DROP TABLE IF EXISTS quizgame.usage_daily;
DROP TABLE IF EXISTS quizgame.review_queue;
DROP TABLE IF EXISTS quizgame.org_settings;

ALTER TABLE settings.platform_app_settings
    DROP CONSTRAINT IF EXISTS platform_iq_guest_join_policy_chk,
    DROP CONSTRAINT IF EXISTS platform_iq_default_mode_chk,
    DROP CONSTRAINT IF EXISTS platform_iq_leaderboard_privacy_chk,
    DROP CONSTRAINT IF EXISTS platform_iq_max_players_chk,
    DROP CONSTRAINT IF EXISTS platform_iq_retention_days_chk;

ALTER TABLE settings.platform_app_settings
    DROP COLUMN IF EXISTS iq_max_concurrent_games,
    DROP COLUMN IF EXISTS iq_max_players_per_game,
    DROP COLUMN IF EXISTS iq_max_kits_per_course,
    DROP COLUMN IF EXISTS iq_retention_days,
    DROP COLUMN IF EXISTS iq_guest_join_policy,
    DROP COLUMN IF EXISTS iq_default_mode,
    DROP COLUMN IF EXISTS iq_default_leaderboard_privacy,
    DROP COLUMN IF EXISTS iq_ai_generation_enabled,
    DROP COLUMN IF EXISTS iq_ai_generations_per_day;
