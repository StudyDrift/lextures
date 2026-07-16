-- VC.4 — Board real-time collaboration: Y.js updates/snapshots + boards_realtime sub-flag.

CREATE TABLE IF NOT EXISTS board.board_updates (
    id         BIGSERIAL PRIMARY KEY,
    board_id   UUID NOT NULL REFERENCES board.boards (id) ON DELETE CASCADE,
    author_id  UUID REFERENCES "user".users (id) ON DELETE SET NULL,
    update     BYTEA NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_board_updates_board ON board.board_updates (board_id, created_at);

COMMENT ON TABLE board.board_updates IS
    'VC.4: Append-only Y.js sync updates for collaboration board CRDT state.';

CREATE TABLE IF NOT EXISTS board.board_snapshots (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    board_id   UUID NOT NULL REFERENCES board.boards (id) ON DELETE CASCADE,
    state      BYTEA NOT NULL,
    taken_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_board_snapshots_board ON board.board_snapshots (board_id, taken_at);

COMMENT ON TABLE board.board_snapshots IS
    'VC.4: Compacted Y.js document snapshots for collaboration boards.';

ALTER TABLE settings.platform_app_settings
    ADD COLUMN IF NOT EXISTS ff_boards_realtime BOOLEAN;

COMMENT ON COLUMN settings.platform_app_settings.ff_boards_realtime IS
    'VC.4: Platform sub-flag for board WebSocket realtime sync. Default OFF; requires ff_visual_boards.';
