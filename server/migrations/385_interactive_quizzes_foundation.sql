-- IQ.1 — Interactive Quizzes foundation: per-course flag, quizgame schema, platform master flag.

ALTER TABLE course.courses
  ADD COLUMN IF NOT EXISTS interactive_quizzes_enabled BOOLEAN NOT NULL DEFAULT FALSE;

COMMENT ON COLUMN course.courses.interactive_quizzes_enabled IS
    'IQ.1: Enables live game-based quizzes for this course. Default off.';

CREATE SCHEMA IF NOT EXISTS quizgame;

DO $$ BEGIN
    CREATE TYPE quizgame.kit_visibility AS ENUM ('private', 'course', 'org', 'public');
EXCEPTION
    WHEN duplicate_object THEN NULL;
END $$;

DO $$ BEGIN
    CREATE TYPE quizgame.kit_status AS ENUM ('draft', 'ready', 'archived');
EXCEPTION
    WHEN duplicate_object THEN NULL;
END $$;

CREATE TABLE IF NOT EXISTS quizgame.kits (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    course_id       UUID NOT NULL REFERENCES course.courses (id) ON DELETE CASCADE,
    title           TEXT NOT NULL,
    description     TEXT NOT NULL DEFAULT '',
    slug            TEXT NOT NULL,
    cover_image_ref TEXT,
    status          quizgame.kit_status     NOT NULL DEFAULT 'draft',
    visibility      quizgame.kit_visibility NOT NULL DEFAULT 'course',
    tags            TEXT[] NOT NULL DEFAULT '{}',
    question_count  INTEGER NOT NULL DEFAULT 0,
    archived        BOOLEAN NOT NULL DEFAULT FALSE,
    created_by      UUID REFERENCES "user".users (id) ON DELETE SET NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (course_id, slug)
);

CREATE INDEX IF NOT EXISTS idx_quizgame_kits_course ON quizgame.kits (course_id) WHERE archived = FALSE;
CREATE INDEX IF NOT EXISTS idx_quizgame_kits_tags ON quizgame.kits USING gin (tags);

COMMENT ON TABLE quizgame.kits IS
    'IQ.1: Course-scoped live quiz kits. Questions and hosting land in later IQ stories.';

ALTER TABLE settings.platform_app_settings
    ADD COLUMN IF NOT EXISTS ff_interactive_quizzes BOOLEAN;

COMMENT ON COLUMN settings.platform_app_settings.ff_interactive_quizzes IS
    'IQ.1: Platform master switch for Live Quizzes. Default OFF.';
