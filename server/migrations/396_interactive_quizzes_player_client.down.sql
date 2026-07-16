ALTER TABLE quizgame.session_players
    DROP COLUMN IF EXISTS client_meta,
    DROP COLUMN IF EXISTS last_seen_at;
