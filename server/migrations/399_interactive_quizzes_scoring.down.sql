DROP TABLE IF EXISTS quizgame.player_powerups;

ALTER TABLE quizgame.session_responses
    DROP COLUMN IF EXISTS points_breakdown;

ALTER TABLE quizgame.sessions
    DROP COLUMN IF EXISTS leaderboard_privacy,
    DROP COLUMN IF EXISTS scoring_config,
    DROP COLUMN IF EXISTS scoring_profile_ver,
    DROP COLUMN IF EXISTS scoring_profile;
