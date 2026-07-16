-- VC.6 rollback — board access control.

ALTER TABLE settings.platform_app_settings
    DROP COLUMN IF EXISTS ff_boards_external_sharing;

DROP TABLE IF EXISTS board.board_shares;
DROP TABLE IF EXISTS board.board_members;

ALTER TABLE board.posts
    DROP COLUMN IF EXISTS guest_display_name;

ALTER TABLE board.boards
    DROP CONSTRAINT IF EXISTS board_boards_visibility_check,
    DROP CONSTRAINT IF EXISTS board_boards_attribution_check;

ALTER TABLE board.boards
    DROP COLUMN IF EXISTS visibility,
    DROP COLUMN IF EXISTS visibility_target,
    DROP COLUMN IF EXISTS attribution,
    DROP COLUMN IF EXISTS can_post,
    DROP COLUMN IF EXISTS can_interact,
    DROP COLUMN IF EXISTS can_arrange;
