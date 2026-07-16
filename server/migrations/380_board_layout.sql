-- VC.3 — Board layouts, sections, and arrangement fields.

ALTER TABLE board.boards
  ADD COLUMN IF NOT EXISTS layout TEXT NOT NULL DEFAULT 'wall',
  ADD COLUMN IF NOT EXISTS layout_locked BOOLEAN NOT NULL DEFAULT FALSE,
  ADD COLUMN IF NOT EXISTS settings JSONB NOT NULL DEFAULT '{}';

ALTER TABLE board.boards
  DROP CONSTRAINT IF EXISTS board_boards_layout_check;
ALTER TABLE board.boards
  ADD CONSTRAINT board_boards_layout_check
    CHECK (layout IN ('wall', 'stream', 'grid', 'columns', 'canvas', 'timeline', 'map'));

COMMENT ON COLUMN board.boards.layout IS
    'VC.3: Active layout mode (wall|stream|grid|columns|canvas|timeline|map).';
COMMENT ON COLUMN board.boards.layout_locked IS
    'VC.3: When true, non-managers may post but not rearrange cards.';
COMMENT ON COLUMN board.boards.settings IS
    'VC.3: Per-layout options (map center/zoom, timeline axis range, …).';

CREATE TABLE IF NOT EXISTS board.sections (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    board_id    UUID NOT NULL REFERENCES board.boards (id) ON DELETE CASCADE,
    title       TEXT NOT NULL,
    sort_index  DOUBLE PRECISION NOT NULL DEFAULT 0,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_sections_board ON board.sections (board_id);

COMMENT ON TABLE board.sections IS
    'VC.3: Named columns/sections for the columns (shelf) layout.';

ALTER TABLE board.posts
  ADD COLUMN IF NOT EXISTS event_date TIMESTAMPTZ,
  ADD COLUMN IF NOT EXISTS lat DOUBLE PRECISION,
  ADD COLUMN IF NOT EXISTS lng DOUBLE PRECISION;

COMMENT ON COLUMN board.posts.event_date IS 'VC.3: Timeline layout date for the card.';
COMMENT ON COLUMN board.posts.lat IS 'VC.3: Map layout latitude (−90…90).';
COMMENT ON COLUMN board.posts.lng IS 'VC.3: Map layout longitude (−180…180).';

-- FK now that sections exists (section_id was a bare UUID in VC.2).
DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1 FROM pg_constraint WHERE conname = 'fk_posts_section'
  ) THEN
    ALTER TABLE board.posts
      ADD CONSTRAINT fk_posts_section FOREIGN KEY (section_id)
      REFERENCES board.sections (id) ON DELETE SET NULL;
  END IF;
END $$;
