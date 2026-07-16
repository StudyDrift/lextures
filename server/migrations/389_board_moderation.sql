-- VC.7 — Board moderation, safety, and content governance.

ALTER TABLE board.boards
  ADD COLUMN IF NOT EXISTS moderation_mode TEXT NOT NULL DEFAULT 'open',
  ADD COLUMN IF NOT EXISTS filter_action TEXT NOT NULL DEFAULT 'flag',
  ADD COLUMN IF NOT EXISTS locked BOOLEAN NOT NULL DEFAULT FALSE,
  ADD COLUMN IF NOT EXISTS frozen_until TIMESTAMPTZ;

ALTER TABLE board.boards
  DROP CONSTRAINT IF EXISTS board_boards_moderation_mode_check;
ALTER TABLE board.boards
  ADD CONSTRAINT board_boards_moderation_mode_check
    CHECK (moderation_mode IN ('open', 'approval'));

ALTER TABLE board.boards
  DROP CONSTRAINT IF EXISTS board_boards_filter_action_check;
ALTER TABLE board.boards
  ADD CONSTRAINT board_boards_filter_action_check
    CHECK (filter_action IN ('block', 'flag'));

COMMENT ON COLUMN board.boards.moderation_mode IS
    'VC.7: open|approval. In approval, non-manager posts start as pending.';
COMMENT ON COLUMN board.boards.filter_action IS
    'VC.7: block|flag. How content-filter matches are handled on write.';
COMMENT ON COLUMN board.boards.locked IS
    'VC.7: When true, board is fully read-only for non-managers.';
COMMENT ON COLUMN board.boards.frozen_until IS
    'VC.7: When set and in the future, posting is frozen for non-managers.';

ALTER TABLE board.posts
  ADD COLUMN IF NOT EXISTS status TEXT NOT NULL DEFAULT 'approved',
  ADD COLUMN IF NOT EXISTS hidden BOOLEAN NOT NULL DEFAULT FALSE,
  ADD COLUMN IF NOT EXISTS removed BOOLEAN NOT NULL DEFAULT FALSE;

ALTER TABLE board.posts
  DROP CONSTRAINT IF EXISTS board_posts_status_check;
ALTER TABLE board.posts
  ADD CONSTRAINT board_posts_status_check
    CHECK (status IN ('approved', 'pending', 'rejected'));

CREATE INDEX IF NOT EXISTS idx_posts_board_status ON board.posts (board_id, status);
CREATE INDEX IF NOT EXISTS idx_posts_board_hidden ON board.posts (board_id, hidden)
  WHERE hidden = TRUE OR removed = TRUE;

COMMENT ON COLUMN board.posts.status IS
    'VC.7: approved|pending|rejected. Pending posts are invisible to peers.';
COMMENT ON COLUMN board.posts.hidden IS
    'VC.7: Soft-hidden by a manager; invisible to peers, retained for audit.';
COMMENT ON COLUMN board.posts.removed IS
    'VC.7: Soft-removed by a manager; invisible to peers, retained for audit.';

CREATE TABLE IF NOT EXISTS board.reports (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    board_id     UUID NOT NULL REFERENCES board.boards (id) ON DELETE CASCADE,
    post_id      UUID REFERENCES board.posts (id) ON DELETE CASCADE,
    comment_id   UUID REFERENCES board.post_comments (id) ON DELETE CASCADE,
    reporter_id  UUID REFERENCES "user".users (id) ON DELETE SET NULL,
    reason       TEXT NOT NULL DEFAULT '',
    kind         TEXT NOT NULL DEFAULT 'user',
    status       TEXT NOT NULL DEFAULT 'open',
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    resolved_at  TIMESTAMPTZ,
    resolved_by  UUID REFERENCES "user".users (id) ON DELETE SET NULL,
    CONSTRAINT board_reports_target_check
        CHECK (post_id IS NOT NULL OR comment_id IS NOT NULL),
    CONSTRAINT board_reports_kind_check
        CHECK (kind IN ('user', 'filter', 'av_blocked')),
    CONSTRAINT board_reports_status_check
        CHECK (status IN ('open', 'resolved', 'dismissed'))
);

CREATE INDEX IF NOT EXISTS idx_reports_board_status ON board.reports (board_id, status);
CREATE UNIQUE INDEX IF NOT EXISTS idx_reports_open_post_reporter
    ON board.reports (board_id, post_id, reporter_id)
    WHERE status = 'open' AND post_id IS NOT NULL AND reporter_id IS NOT NULL AND kind = 'user';
CREATE UNIQUE INDEX IF NOT EXISTS idx_reports_open_comment_reporter
    ON board.reports (board_id, comment_id, reporter_id)
    WHERE status = 'open' AND comment_id IS NOT NULL AND reporter_id IS NOT NULL AND kind = 'user';

COMMENT ON TABLE board.reports IS
    'VC.7: User reports, filter flags, and AV-blocked attachment flags for moderation queue.';

CREATE TABLE IF NOT EXISTS board.moderation_log (
    id          BIGSERIAL PRIMARY KEY,
    board_id    UUID NOT NULL REFERENCES board.boards (id) ON DELETE CASCADE,
    actor_id    UUID REFERENCES "user".users (id) ON DELETE SET NULL,
    action      TEXT NOT NULL,
    target_type TEXT NOT NULL,
    target_id   UUID,
    reason      TEXT NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT board_modlog_target_type_check
        CHECK (target_type IN ('post', 'comment', 'board', 'report'))
);

CREATE INDEX IF NOT EXISTS idx_modlog_board ON board.moderation_log (board_id, created_at DESC);

COMMENT ON TABLE board.moderation_log IS
    'VC.7: Append-only audit trail of moderation actions (approve/reject/hide/remove/lock/freeze/…).';
