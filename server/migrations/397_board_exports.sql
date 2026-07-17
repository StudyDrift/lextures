-- VC.9 — Board export jobs (PDF / CSV / image).

CREATE TABLE IF NOT EXISTS board.export_jobs (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    board_id     UUID NOT NULL REFERENCES board.boards (id) ON DELETE CASCADE,
    format       TEXT NOT NULL,
    status       TEXT NOT NULL DEFAULT 'pending',
    storage_key  TEXT,
    error        TEXT NOT NULL DEFAULT '',
    include_moderation BOOLEAN NOT NULL DEFAULT FALSE,
    requested_by UUID REFERENCES "user".users (id) ON DELETE SET NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ,
    CONSTRAINT board_export_jobs_format_check
        CHECK (format IN ('pdf', 'csv', 'image')),
    CONSTRAINT board_export_jobs_status_check
        CHECK (status IN ('pending', 'running', 'done', 'failed'))
);

CREATE INDEX IF NOT EXISTS idx_export_jobs_board
    ON board.export_jobs (board_id, created_at DESC);

COMMENT ON TABLE board.export_jobs IS
    'VC.9: Async board export jobs (PDF/CSV/image) with object-store keys.';
