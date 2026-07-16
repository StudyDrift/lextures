-- IQ.2 — Quiz-kit questions: types, ordered items, bank link, question_count trigger.

DO $$ BEGIN
    CREATE TYPE quizgame.question_type AS ENUM (
        'mc_single', 'mc_multiple', 'true_false', 'type_answer',
        'numeric', 'poll', 'ordering', 'word_cloud'
    );
EXCEPTION
    WHEN duplicate_object THEN NULL;
END $$;

DO $$ BEGIN
    CREATE TYPE quizgame.points_style AS ENUM ('standard', 'double', 'no_points');
EXCEPTION
    WHEN duplicate_object THEN NULL;
END $$;

CREATE TABLE IF NOT EXISTS quizgame.questions (
    id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    kit_id             UUID NOT NULL REFERENCES quizgame.kits (id) ON DELETE CASCADE,
    position           INTEGER NOT NULL,
    question_type      quizgame.question_type NOT NULL DEFAULT 'mc_single',
    prompt             TEXT NOT NULL DEFAULT '',
    prompt_media_ref   TEXT,
    prompt_media_alt   TEXT,
    options            JSONB NOT NULL DEFAULT '[]'::jsonb,
    correct_answer     JSONB,
    time_limit_seconds INTEGER NOT NULL DEFAULT 20
        CHECK (time_limit_seconds BETWEEN 5 AND 240),
    points_style       quizgame.points_style NOT NULL DEFAULT 'standard',
    answer_shuffle     BOOLEAN NOT NULL DEFAULT TRUE,
    explanation        TEXT,
    source_question_id UUID REFERENCES course.questions (id) ON DELETE SET NULL,
    version            INTEGER NOT NULL DEFAULT 1,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (kit_id, position)
);

CREATE INDEX IF NOT EXISTS idx_quizgame_questions_kit ON quizgame.questions (kit_id, position);
CREATE INDEX IF NOT EXISTS idx_quizgame_questions_source ON quizgame.questions (source_question_id)
    WHERE source_question_id IS NOT NULL;

COMMENT ON TABLE quizgame.questions IS
    'IQ.2: Ordered game questions within a quiz kit. Bank link is copy-with-link (non-blocking).';

CREATE OR REPLACE FUNCTION quizgame.sync_kit_question_count() RETURNS TRIGGER AS $$
BEGIN
    IF TG_OP = 'INSERT' THEN
        UPDATE quizgame.kits
        SET question_count = (
                SELECT COUNT(*)::integer FROM quizgame.questions WHERE kit_id = NEW.kit_id
            ),
            updated_at = NOW()
        WHERE id = NEW.kit_id;
        RETURN NEW;
    ELSIF TG_OP = 'DELETE' THEN
        UPDATE quizgame.kits
        SET question_count = (
                SELECT COUNT(*)::integer FROM quizgame.questions WHERE kit_id = OLD.kit_id
            ),
            updated_at = NOW()
        WHERE id = OLD.kit_id;
        RETURN OLD;
    END IF;
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_quizgame_questions_count ON quizgame.questions;
CREATE TRIGGER trg_quizgame_questions_count
    AFTER INSERT OR DELETE ON quizgame.questions
    FOR EACH ROW EXECUTE FUNCTION quizgame.sync_kit_question_count();
