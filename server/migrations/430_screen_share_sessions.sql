-- SS.1 — Full-screen sharing: schema, per-course flag, platform master flag.

CREATE SCHEMA IF NOT EXISTS screenshare;

DO $$ BEGIN
    CREATE TYPE screenshare.session_status AS ENUM ('open','presenting','ended','abandoned');
EXCEPTION
    WHEN duplicate_object THEN NULL;
END $$;

DO $$ BEGIN
    CREATE TYPE screenshare.present_policy AS ENUM ('host_only','request','free_for_all');
EXCEPTION
    WHEN duplicate_object THEN NULL;
END $$;

DO $$ BEGIN
    CREATE TYPE screenshare.participant_role AS ENUM ('host','presenter','viewer','display');
EXCEPTION
    WHEN duplicate_object THEN NULL;
END $$;

CREATE TABLE IF NOT EXISTS screenshare.sessions (
  id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  course_id      UUID NOT NULL REFERENCES course.courses (id) ON DELETE CASCADE,
  host_id        UUID REFERENCES "user".users (id) ON DELETE SET NULL,
  title          TEXT,
  status         screenshare.session_status NOT NULL DEFAULT 'open',
  policy         screenshare.present_policy NOT NULL DEFAULT 'request',
  present_audio  BOOLEAN NOT NULL DEFAULT FALSE,
  viewer_cap     INTEGER NOT NULL DEFAULT 50,
  active_presenter_id UUID REFERENCES "user".users (id) ON DELETE SET NULL,
  settings       JSONB NOT NULL DEFAULT '{}'::jsonb,
  join_token_hash TEXT NOT NULL,
  started_at     TIMESTAMPTZ,
  ended_at       TIMESTAMPTZ,
  created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_screenshare_sessions_course
  ON screenshare.sessions (course_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_screenshare_sessions_active
  ON screenshare.sessions (course_id) WHERE status IN ('open','presenting');

CREATE INDEX IF NOT EXISTS idx_screenshare_sessions_reaper
  ON screenshare.sessions (status, created_at)
  WHERE status IN ('open','presenting');

COMMENT ON TABLE screenshare.sessions IS
  'SS.1: Screen-share sessions. Metadata only — no media frames are stored.';

CREATE TABLE IF NOT EXISTS screenshare.participants (
  id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  session_id   UUID NOT NULL REFERENCES screenshare.sessions (id) ON DELETE CASCADE,
  user_id      UUID REFERENCES "user".users (id) ON DELETE SET NULL,
  role         screenshare.participant_role NOT NULL,
  connected    BOOLEAN NOT NULL DEFAULT TRUE,
  joined_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  left_at      TIMESTAMPTZ,
  UNIQUE (session_id, user_id, role)
);

CREATE INDEX IF NOT EXISTS idx_screenshare_participants_session
  ON screenshare.participants (session_id);

COMMENT ON TABLE screenshare.participants IS
  'SS.1: Participants in a screen-share session. user_id NULL only for anon display links.';

CREATE TABLE IF NOT EXISTS screenshare.events (
  id         BIGSERIAL PRIMARY KEY,
  session_id UUID NOT NULL REFERENCES screenshare.sessions (id) ON DELETE CASCADE,
  seq        INTEGER NOT NULL,
  type       TEXT NOT NULL,
  actor_id   UUID REFERENCES "user".users (id) ON DELETE SET NULL,
  payload    JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (session_id, seq)
);

CREATE INDEX IF NOT EXISTS idx_screenshare_events_session
  ON screenshare.events (session_id, seq);

COMMENT ON TABLE screenshare.events IS
  'SS.1: Append-only audit/event log. Never stores media frames.';

ALTER TABLE course.courses
  ADD COLUMN IF NOT EXISTS screen_share_enabled BOOLEAN NOT NULL DEFAULT FALSE;

COMMENT ON COLUMN course.courses.screen_share_enabled IS
  'SS.1: Enables cableless screen sharing for this course. Default off.';

ALTER TABLE settings.platform_app_settings
  ADD COLUMN IF NOT EXISTS ff_screen_share BOOLEAN;

COMMENT ON COLUMN settings.platform_app_settings.ff_screen_share IS
  'SS.1: Platform master switch for Screen Sharing. Default OFF.';
