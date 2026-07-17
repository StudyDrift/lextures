-- IQ.7 — Post-game reports, gradebook links, and gradebook-push sub-flag.

CREATE TABLE IF NOT EXISTS quizgame.game_reports (
    session_id     UUID PRIMARY KEY REFERENCES quizgame.sessions (id) ON DELETE CASCADE,
    player_count   INTEGER NOT NULL,
    answered_count INTEGER NOT NULL,
    score_avg      NUMERIC(10, 2),
    score_median   NUMERIC(10, 2),
    score_max      INTEGER,
    per_question   JSONB NOT NULL DEFAULT '[]'::jsonb,
    generated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE quizgame.game_reports IS
    'IQ.7: Cached post-game report aggregates; recomputable from session_responses.';
COMMENT ON COLUMN quizgame.game_reports.per_question IS
    'IQ.7: [{index, prompt, correctPct, avgMs, distribution, hardestRank, sourceQuestionId}]';

CREATE TABLE IF NOT EXISTS quizgame.gradebook_links (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id        UUID REFERENCES quizgame.sessions (id) ON DELETE CASCADE,
    assignment_id     UUID REFERENCES quizgame.assignments (id) ON DELETE CASCADE,
    course_id         UUID NOT NULL REFERENCES course.courses (id) ON DELETE CASCADE,
    gradebook_item_id UUID NOT NULL,
    mapping           TEXT NOT NULL DEFAULT 'participation',
    points_possible   NUMERIC(6, 2),
    participation_pct NUMERIC(5, 2) NOT NULL DEFAULT 50.0,
    created_by        UUID REFERENCES "user".users (id) ON DELETE SET NULL,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT quizgame_gradebook_links_target_chk
        CHECK (session_id IS NOT NULL OR assignment_id IS NOT NULL),
    CONSTRAINT quizgame_gradebook_links_mapping_chk
        CHECK (mapping IN ('raw_points', 'percent_correct', 'participation'))
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_quizgame_gradebook_links_session
    ON quizgame.gradebook_links (session_id)
    WHERE session_id IS NOT NULL;

CREATE UNIQUE INDEX IF NOT EXISTS idx_quizgame_gradebook_links_assignment
    ON quizgame.gradebook_links (assignment_id)
    WHERE assignment_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_quizgame_gradebook_links_course
    ON quizgame.gradebook_links (course_id);

COMMENT ON TABLE quizgame.gradebook_links IS
    'IQ.7: Idempotent linkage from a game/assignment to a coursegrades structure item.';

ALTER TABLE settings.platform_app_settings
    ADD COLUMN IF NOT EXISTS ff_iq_gradebook_push BOOLEAN;

COMMENT ON COLUMN settings.platform_app_settings.ff_iq_gradebook_push IS
    'IQ.7: Enable pushing live-quiz scores into the course gradebook. Requires ff_interactive_quizzes.';
