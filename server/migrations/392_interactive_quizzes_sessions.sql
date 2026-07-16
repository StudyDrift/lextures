-- IQ.3 — Live game hosting: sessions, players, responses, event log + iq_live_hosting sub-flag.

DO $$ BEGIN
    CREATE TYPE quizgame.session_status AS ENUM ('lobby', 'running', 'paused', 'ended', 'abandoned');
EXCEPTION
    WHEN duplicate_object THEN NULL;
END $$;

DO $$ BEGIN
    CREATE TYPE quizgame.session_mode AS ENUM ('live_classic', 'team', 'student_paced', 'homework');
EXCEPTION
    WHEN duplicate_object THEN NULL;
END $$;

CREATE TABLE IF NOT EXISTS quizgame.sessions (
    id                   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    kit_id               UUID NOT NULL REFERENCES quizgame.kits (id) ON DELETE RESTRICT,
    course_id            UUID NOT NULL REFERENCES course.courses (id) ON DELETE CASCADE,
    host_id              UUID REFERENCES "user".users (id) ON DELETE SET NULL,
    join_code            TEXT,
    mode                 quizgame.session_mode   NOT NULL DEFAULT 'live_classic',
    status               quizgame.session_status NOT NULL DEFAULT 'lobby',
    pacing               TEXT NOT NULL DEFAULT 'manual',
    kit_snapshot         JSONB NOT NULL,
    current_index        INTEGER NOT NULL DEFAULT -1,
    current_phase        TEXT NOT NULL DEFAULT 'lobby',
    question_opened_at   TIMESTAMPTZ,
    question_deadline_at TIMESTAMPTZ,
    host_disconnected_at TIMESTAMPTZ,
    settings             JSONB NOT NULL DEFAULT '{}'::jsonb,
    started_at           TIMESTAMPTZ,
    ended_at             TIMESTAMPTZ,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_quizgame_active_join_code
    ON quizgame.sessions (join_code)
    WHERE join_code IS NOT NULL AND status IN ('lobby', 'running', 'paused');

CREATE INDEX IF NOT EXISTS idx_quizgame_sessions_course
    ON quizgame.sessions (course_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_quizgame_sessions_reaper
    ON quizgame.sessions (status, host_disconnected_at)
    WHERE status IN ('lobby', 'running', 'paused');

COMMENT ON TABLE quizgame.sessions IS
    'IQ.3: Authoritative live quiz game sessions. kit_snapshot freezes questions at start.';

CREATE TABLE IF NOT EXISTS quizgame.session_players (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id   UUID NOT NULL REFERENCES quizgame.sessions (id) ON DELETE CASCADE,
    user_id      UUID REFERENCES "user".users (id) ON DELETE SET NULL,
    nickname     TEXT NOT NULL,
    team_id      UUID,
    player_token TEXT NOT NULL,
    total_score  INTEGER NOT NULL DEFAULT 0,
    streak       INTEGER NOT NULL DEFAULT 0,
    connected    BOOLEAN NOT NULL DEFAULT TRUE,
    joined_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    removed_at   TIMESTAMPTZ,
    UNIQUE (session_id, nickname)
);

CREATE INDEX IF NOT EXISTS idx_quizgame_players_session
    ON quizgame.session_players (session_id);

CREATE UNIQUE INDEX IF NOT EXISTS idx_quizgame_players_token
    ON quizgame.session_players (player_token);

COMMENT ON TABLE quizgame.session_players IS
    'IQ.3: Players in a live quiz session. player_token is a reconnect secret (hashed).';

CREATE TABLE IF NOT EXISTS quizgame.session_responses (
    session_id     UUID NOT NULL REFERENCES quizgame.sessions (id) ON DELETE CASCADE,
    question_index INTEGER NOT NULL,
    player_id      UUID NOT NULL REFERENCES quizgame.session_players (id) ON DELETE CASCADE,
    answer         JSONB NOT NULL,
    is_correct     BOOLEAN NOT NULL,
    response_ms    INTEGER NOT NULL,
    points         INTEGER NOT NULL DEFAULT 0,
    answered_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (session_id, question_index, player_id)
);

COMMENT ON TABLE quizgame.session_responses IS
    'IQ.3: Idempotent per-player answers. response_ms is server-measured from question open.';

CREATE TABLE IF NOT EXISTS quizgame.session_events (
    id         BIGSERIAL PRIMARY KEY,
    session_id UUID NOT NULL REFERENCES quizgame.sessions (id) ON DELETE CASCADE,
    seq        INTEGER NOT NULL,
    type       TEXT NOT NULL,
    payload    JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (session_id, seq)
);

CREATE INDEX IF NOT EXISTS idx_quizgame_events_session
    ON quizgame.session_events (session_id, seq);

COMMENT ON TABLE quizgame.session_events IS
    'IQ.3: Append-only event log for reconnect replay and audit.';

ALTER TABLE settings.platform_app_settings
    ADD COLUMN IF NOT EXISTS ff_iq_live_hosting BOOLEAN;

COMMENT ON COLUMN settings.platform_app_settings.ff_iq_live_hosting IS
    'IQ.3: Platform sub-flag for live game hosting engine. Default OFF; requires ff_interactive_quizzes.';
