-- VC.6 — Board visibility, contributor policy, members, share links, external-sharing flag.

ALTER TABLE board.boards
  ADD COLUMN IF NOT EXISTS visibility TEXT NOT NULL DEFAULT 'course',
  ADD COLUMN IF NOT EXISTS visibility_target UUID,
  ADD COLUMN IF NOT EXISTS attribution TEXT NOT NULL DEFAULT 'named',
  ADD COLUMN IF NOT EXISTS can_post BOOLEAN NOT NULL DEFAULT TRUE,
  ADD COLUMN IF NOT EXISTS can_interact BOOLEAN NOT NULL DEFAULT TRUE,
  ADD COLUMN IF NOT EXISTS can_arrange BOOLEAN NOT NULL DEFAULT FALSE;

ALTER TABLE board.boards
  DROP CONSTRAINT IF EXISTS board_boards_visibility_check;
ALTER TABLE board.boards
  ADD CONSTRAINT board_boards_visibility_check
    CHECK (visibility IN ('course', 'section', 'group', 'invite', 'link', 'public'));

ALTER TABLE board.boards
  DROP CONSTRAINT IF EXISTS board_boards_attribution_check;
ALTER TABLE board.boards
  ADD CONSTRAINT board_boards_attribution_check
    CHECK (attribution IN ('named', 'anon_to_peers', 'anonymous'));

COMMENT ON COLUMN board.boards.visibility IS
    'VC.6: course|section|group|invite|link|public. Default course.';
COMMENT ON COLUMN board.boards.visibility_target IS
    'VC.6: section_id or group_id when visibility is section/group.';
COMMENT ON COLUMN board.boards.attribution IS
    'VC.6: named|anon_to_peers|anonymous. Author id retained for audit; API may omit it.';
COMMENT ON COLUMN board.boards.can_post IS
    'VC.6: Default contributor policy — in-scope members may create posts.';
COMMENT ON COLUMN board.boards.can_interact IS
    'VC.6: Default contributor policy — in-scope members may react/comment.';
COMMENT ON COLUMN board.boards.can_arrange IS
    'VC.6: Default contributor policy — in-scope members may rearrange cards.';

ALTER TABLE board.posts
  ADD COLUMN IF NOT EXISTS guest_display_name TEXT NOT NULL DEFAULT '';

COMMENT ON COLUMN board.posts.guest_display_name IS
    'VC.6: Display name for unauthenticated contribute-link authors (author_id NULL).';

CREATE TABLE IF NOT EXISTS board.board_members (
    board_id   UUID NOT NULL REFERENCES board.boards (id) ON DELETE CASCADE,
    user_id    UUID NOT NULL REFERENCES "user".users (id) ON DELETE CASCADE,
    role       TEXT NOT NULL DEFAULT 'contributor',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (board_id, user_id),
    CONSTRAINT board_members_role_check
        CHECK (role IN ('owner', 'editor', 'contributor', 'viewer'))
);

CREATE INDEX IF NOT EXISTS idx_board_members_user ON board.board_members (user_id);

COMMENT ON TABLE board.board_members IS
    'VC.6: Explicit members for invite-only boards (plus per-member role).';

CREATE TABLE IF NOT EXISTS board.board_shares (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    board_id      UUID NOT NULL REFERENCES board.boards (id) ON DELETE CASCADE,
    token_hash    TEXT NOT NULL UNIQUE,
    capability    TEXT NOT NULL DEFAULT 'view',
    password_hash TEXT,
    expires_at    TIMESTAMPTZ,
    created_by    UUID REFERENCES "user".users (id) ON DELETE SET NULL,
    revoked_at    TIMESTAMPTZ,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT board_shares_capability_check
        CHECK (capability IN ('view', 'contribute'))
);

CREATE INDEX IF NOT EXISTS idx_board_shares_board ON board.board_shares (board_id);

COMMENT ON TABLE board.board_shares IS
    'VC.6: Capability share links. token_hash is SHA-256 of the raw URL token; password_hash is Argon2id.';

ALTER TABLE settings.platform_app_settings
    ADD COLUMN IF NOT EXISTS ff_boards_external_sharing BOOLEAN;

COMMENT ON COLUMN settings.platform_app_settings.ff_boards_external_sharing IS
    'VC.6: Allow link/public board visibility and share links. Default OFF; requires ff_visual_boards.';
