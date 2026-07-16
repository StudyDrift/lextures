-- IQ.4 — Player client metadata for reconnect support (no fingerprinting).

ALTER TABLE quizgame.session_players
    ADD COLUMN IF NOT EXISTS last_seen_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS client_meta JSONB NOT NULL DEFAULT '{}'::jsonb;

COMMENT ON COLUMN quizgame.session_players.last_seen_at IS
    'IQ.4: Last WS activity / join touch; used for support reconnect diagnostics.';
COMMENT ON COLUMN quizgame.session_players.client_meta IS
    'IQ.4: Coarse device/browser hints (no fingerprinting).';
