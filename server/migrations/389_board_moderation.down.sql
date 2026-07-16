DROP TABLE IF EXISTS board.moderation_log;
DROP TABLE IF EXISTS board.reports;

DROP INDEX IF EXISTS board.idx_posts_board_hidden;
DROP INDEX IF EXISTS board.idx_posts_board_status;

ALTER TABLE board.posts
  DROP CONSTRAINT IF EXISTS board_posts_status_check;
ALTER TABLE board.posts
  DROP COLUMN IF EXISTS removed,
  DROP COLUMN IF EXISTS hidden,
  DROP COLUMN IF EXISTS status;

ALTER TABLE board.boards
  DROP CONSTRAINT IF EXISTS board_boards_filter_action_check;
ALTER TABLE board.boards
  DROP CONSTRAINT IF EXISTS board_boards_moderation_mode_check;
ALTER TABLE board.boards
  DROP COLUMN IF EXISTS frozen_until,
  DROP COLUMN IF EXISTS locked,
  DROP COLUMN IF EXISTS filter_action,
  DROP COLUMN IF EXISTS moderation_mode;
