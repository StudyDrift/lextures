-- IQ.9 — Moderation, safety, accessibility & fair play.

ALTER TABLE quizgame.sessions
    ADD COLUMN IF NOT EXISTS allow_guests      BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS lobby_locked      BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS names_muted       BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS one_session_rule  TEXT NOT NULL DEFAULT 'takeover',
    ADD COLUMN IF NOT EXISTS max_joins_per_ip  INTEGER NOT NULL DEFAULT 5;

DO $$ BEGIN
    ALTER TABLE quizgame.sessions
        ADD CONSTRAINT quizgame_sessions_one_session_rule_chk
        CHECK (one_session_rule IN ('takeover', 'refuse', 'off'));
EXCEPTION
    WHEN duplicate_object THEN NULL;
END $$;

COMMENT ON COLUMN quizgame.sessions.allow_guests IS
    'IQ.9: When true (and platform guest-join flag on), unauthenticated players may join.';
COMMENT ON COLUMN quizgame.sessions.lobby_locked IS
    'IQ.9: Host lock — no new joins while true.';
COMMENT ON COLUMN quizgame.sessions.names_muted IS
    'IQ.9: Projector/public surfaces show Player N instead of nicknames.';
COMMENT ON COLUMN quizgame.sessions.one_session_rule IS
    'IQ.9: takeover | refuse | off — concurrent join by same enrolled identity.';
COMMENT ON COLUMN quizgame.sessions.max_joins_per_ip IS
    'IQ.9: Cap on distinct player rows per salted join_ip_hash for this game.';

ALTER TABLE quizgame.session_players
    ADD COLUMN IF NOT EXISTS banned           BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS renamed_by_host  BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS join_ip_hash     TEXT;

COMMENT ON COLUMN quizgame.session_players.banned IS
    'IQ.9: Host kick/ban — blocks rejoin for this game (enrolled by user_id; guests by IP hash).';
COMMENT ON COLUMN quizgame.session_players.renamed_by_host IS
    'IQ.9: Nickname was force-renamed by host.';
COMMENT ON COLUMN quizgame.session_players.join_ip_hash IS
    'IQ.9: Salted per-game IP hash for join limits / integrity; purged with session.';

CREATE TABLE IF NOT EXISTS quizgame.safety_events (
    id         BIGSERIAL PRIMARY KEY,
    session_id UUID NOT NULL REFERENCES quizgame.sessions (id) ON DELETE CASCADE,
    player_id  UUID REFERENCES quizgame.session_players (id) ON DELETE SET NULL,
    actor_id   UUID REFERENCES "user".users (id) ON DELETE SET NULL,
    kind       TEXT NOT NULL,
    detail     JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_quizgame_safety_session
    ON quizgame.safety_events (session_id, created_at);

COMMENT ON TABLE quizgame.safety_events IS
    'IQ.9: Audit trail for nickname_denied, kicked, banned, renamed, muted, integrity_flag, content_flag.';

ALTER TABLE settings.platform_app_settings
    ADD COLUMN IF NOT EXISTS ff_iq_guest_join BOOLEAN;

COMMENT ON COLUMN settings.platform_app_settings.ff_iq_guest_join IS
    'IQ.9: Platform sub-flag for guest join on live quizzes. Default OFF; requires ff_interactive_quizzes + ff_iq_live_hosting. Blocked for under-13 courses.';
