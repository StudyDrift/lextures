-- VC.1 — Collaboration boards foundation: per-course flag, board schema, platform master flag.

ALTER TABLE course.courses
  ADD COLUMN IF NOT EXISTS visual_boards_enabled BOOLEAN NOT NULL DEFAULT FALSE;

COMMENT ON COLUMN course.courses.visual_boards_enabled IS
    'VC.1: Enables collaboration boards (shared wall) for this course. Default off.';

CREATE SCHEMA IF NOT EXISTS board;

CREATE TABLE IF NOT EXISTS board.boards (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    course_id     UUID NOT NULL REFERENCES course.courses (id) ON DELETE CASCADE,
    title         TEXT NOT NULL,
    description   TEXT NOT NULL DEFAULT '',
    slug          TEXT NOT NULL,
    archived      BOOLEAN NOT NULL DEFAULT FALSE,
    created_by    UUID REFERENCES "user".users (id) ON DELETE SET NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (course_id, slug)
);

CREATE INDEX IF NOT EXISTS idx_boards_course ON board.boards (course_id) WHERE archived = FALSE;

COMMENT ON TABLE board.boards IS
    'VC.1: Course-scoped visual collaboration boards. Posts/layouts land in later VC stories.';

ALTER TABLE settings.platform_app_settings
    ADD COLUMN IF NOT EXISTS ff_visual_boards BOOLEAN;

COMMENT ON COLUMN settings.platform_app_settings.ff_visual_boards IS
    'VC.1: Platform master switch for collaboration boards. Default OFF.';
