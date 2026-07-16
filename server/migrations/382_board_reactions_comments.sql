-- VC.5 — Board reactions, comments, and optional gradebook sync.

ALTER TABLE board.boards
  ADD COLUMN IF NOT EXISTS reaction_mode TEXT NOT NULL DEFAULT 'none',
  ADD COLUMN IF NOT EXISTS assignment_id UUID REFERENCES course.course_structure_items (id) ON DELETE SET NULL;

DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1 FROM pg_constraint WHERE conname = 'board_boards_reaction_mode_check'
  ) THEN
    ALTER TABLE board.boards
      ADD CONSTRAINT board_boards_reaction_mode_check
      CHECK (reaction_mode IN ('none', 'like', 'vote', 'star', 'grade'));
  END IF;
END $$;

COMMENT ON COLUMN board.boards.reaction_mode IS
  'VC.5: Reaction mode for cards — none|like|vote|star|grade.';
COMMENT ON COLUMN board.boards.assignment_id IS
  'VC.5: Optional gradebook structure item for grade sync.';

CREATE TABLE IF NOT EXISTS board.post_reactions (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    post_id    UUID NOT NULL REFERENCES board.posts (id) ON DELETE CASCADE,
    user_id    UUID NOT NULL REFERENCES "user".users (id) ON DELETE CASCADE,
    kind       TEXT NOT NULL,
    value      DOUBLE PRECISION,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT board_post_reactions_kind_check
        CHECK (kind IN ('like', 'vote', 'star', 'grade')),
    CONSTRAINT board_post_reactions_star_value_check
        CHECK (kind <> 'star' OR (value IS NOT NULL AND value >= 1 AND value <= 5)),
    CONSTRAINT board_post_reactions_unique_user_kind
        UNIQUE (post_id, user_id, kind)
);

CREATE INDEX IF NOT EXISTS idx_reactions_post ON board.post_reactions (post_id);

COMMENT ON TABLE board.post_reactions IS
  'VC.5: One reaction of a kind per user per card (like/vote/star/grade).';

CREATE TABLE IF NOT EXISTS board.post_comments (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    post_id    UUID NOT NULL REFERENCES board.posts (id) ON DELETE CASCADE,
    parent_id  UUID REFERENCES board.post_comments (id) ON DELETE CASCADE,
    author_id  UUID REFERENCES "user".users (id) ON DELETE SET NULL,
    body       JSONB NOT NULL,
    hidden     BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_comments_post ON board.post_comments (post_id);

COMMENT ON TABLE board.post_comments IS
  'VC.5: Threaded comments on board cards; hidden soft-hides for audit/FERPA.';
