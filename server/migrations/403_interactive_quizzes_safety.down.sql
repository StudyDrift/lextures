ALTER TABLE settings.platform_app_settings
    DROP COLUMN IF EXISTS ff_iq_guest_join;

DROP TABLE IF EXISTS quizgame.safety_events;

ALTER TABLE quizgame.session_players
    DROP COLUMN IF EXISTS join_ip_hash,
    DROP COLUMN IF EXISTS renamed_by_host,
    DROP COLUMN IF EXISTS banned;

ALTER TABLE quizgame.sessions
    DROP CONSTRAINT IF EXISTS quizgame_sessions_one_session_rule_chk;

ALTER TABLE quizgame.sessions
    DROP COLUMN IF EXISTS max_joins_per_ip,
    DROP COLUMN IF EXISTS one_session_rule,
    DROP COLUMN IF EXISTS names_muted,
    DROP COLUMN IF EXISTS lobby_locked,
    DROP COLUMN IF EXISTS allow_guests;
