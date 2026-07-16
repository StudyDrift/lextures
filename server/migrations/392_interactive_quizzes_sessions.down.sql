-- Reverse IQ.3 live game hosting migration.

ALTER TABLE settings.platform_app_settings
    DROP COLUMN IF EXISTS ff_iq_live_hosting;

DROP TABLE IF EXISTS quizgame.session_events;
DROP TABLE IF EXISTS quizgame.session_responses;
DROP TABLE IF EXISTS quizgame.session_players;
DROP TABLE IF EXISTS quizgame.sessions;

DROP TYPE IF EXISTS quizgame.session_mode;
DROP TYPE IF EXISTS quizgame.session_status;
