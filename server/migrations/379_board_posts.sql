-- VC.2 — Board posts and multi-format content (attachments + layout placeholders for VC.3).

CREATE TABLE IF NOT EXISTS board.post_attachments (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    board_id      UUID NOT NULL REFERENCES board.boards (id) ON DELETE CASCADE,
    storage_key   TEXT NOT NULL,
    file_name     TEXT NOT NULL,
    mime_type     TEXT NOT NULL,
    size_bytes    BIGINT NOT NULL,
    alt_text      TEXT NOT NULL DEFAULT '',
    scan_status   TEXT NOT NULL DEFAULT 'pending', -- pending|clean|blocked
    uploaded_by   UUID REFERENCES "user".users (id) ON DELETE SET NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT board_post_attachments_scan_status_check
        CHECK (scan_status IN ('pending', 'clean', 'blocked'))
);

CREATE INDEX IF NOT EXISTS idx_post_attachments_board ON board.post_attachments (board_id);

COMMENT ON TABLE board.post_attachments IS
    'VC.2: Object-store attachments for board posts (image/file/video/audio).';

CREATE TABLE IF NOT EXISTS board.posts (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    board_id      UUID NOT NULL REFERENCES board.boards (id) ON DELETE CASCADE,
    author_id     UUID REFERENCES "user".users (id) ON DELETE SET NULL,
    content_type  TEXT NOT NULL,
    title         TEXT NOT NULL DEFAULT '',
    body          JSONB,
    link_url      TEXT,
    link_preview  JSONB,
    drawing_data  JSONB,
    attachment_id UUID REFERENCES board.post_attachments (id) ON DELETE SET NULL,
    -- layout columns (owned/used by VC.3)
    section_id    UUID,
    sort_index    DOUBLE PRECISION NOT NULL DEFAULT 0,
    position      JSONB,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT board_posts_content_type_check
        CHECK (content_type IN ('text', 'image', 'file', 'link', 'video', 'audio', 'drawing'))
);

CREATE INDEX IF NOT EXISTS idx_posts_board ON board.posts (board_id);
CREATE INDEX IF NOT EXISTS idx_posts_board_created ON board.posts (board_id, created_at DESC);

COMMENT ON TABLE board.posts IS
    'VC.2: Multi-format cards on a collaboration board. Layout columns reserved for VC.3.';
