ALTER TABLE settings.platform_app_settings
    DROP COLUMN IF EXISTS ff_iq_gradebook_push;

DROP TABLE IF EXISTS quizgame.gradebook_links;
DROP TABLE IF EXISTS quizgame.game_reports;
