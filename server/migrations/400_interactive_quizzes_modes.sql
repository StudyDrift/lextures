-- IQ.6 — Game modes: teams, student-paced player progress, async homework assignments.

CREATE TABLE IF NOT EXISTS quizgame.teams (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id  UUID NOT NULL REFERENCES quizgame.sessions (id) ON DELETE CASCADE,
    name        TEXT NOT NULL,
    color       TEXT,
    total_score INTEGER NOT NULL DEFAULT 0,
    UNIQUE (session_id, name)
);

CREATE INDEX IF NOT EXISTS idx_quizgame_teams_session
    ON quizgame.teams (session_id);

COMMENT ON TABLE quizgame.teams IS
    'IQ.6: Named teams for team-mode games; scores aggregated from member responses.';

-- session_players.team_id was declared in IQ.3 without an FK; attach it now.
DO $$ BEGIN
    ALTER TABLE quizgame.session_players
        ADD CONSTRAINT session_players_team_id_fkey
        FOREIGN KEY (team_id) REFERENCES quizgame.teams (id) ON DELETE SET NULL;
EXCEPTION
    WHEN duplicate_object THEN NULL;
END $$;

-- Per-player progress for student_paced / homework (independent of session clock).
ALTER TABLE quizgame.session_players
    ADD COLUMN IF NOT EXISTS current_index        INTEGER NOT NULL DEFAULT -1,
    ADD COLUMN IF NOT EXISTS current_phase        TEXT NOT NULL DEFAULT 'lobby',
    ADD COLUMN IF NOT EXISTS question_opened_at   TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS question_deadline_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS question_order       JSONB,
    ADD COLUMN IF NOT EXISTS time_budget_ends_at  TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS finished_at          TIMESTAMPTZ;

COMMENT ON COLUMN quizgame.session_players.current_index IS
    'IQ.6: Per-player question index for student_paced/homework (-1 = not started).';
COMMENT ON COLUMN quizgame.session_players.question_order IS
    'IQ.6: Optional shuffled question index order for this player.';

CREATE TABLE IF NOT EXISTS quizgame.assignments (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    kit_id            UUID NOT NULL REFERENCES quizgame.kits (id) ON DELETE RESTRICT,
    course_id         UUID NOT NULL REFERENCES course.courses (id) ON DELETE CASCADE,
    title             TEXT NOT NULL,
    opens_at          TIMESTAMPTZ,
    due_at            TIMESTAMPTZ,
    closes_at         TIMESTAMPTZ,
    attempts_allowed  INTEGER NOT NULL DEFAULT 1,
    grade_policy      TEXT NOT NULL DEFAULT 'best',
    shuffle           BOOLEAN NOT NULL DEFAULT TRUE,
    points_possible   NUMERIC(6, 2),
    gradebook_item_id UUID,
    scoring_profile   TEXT NOT NULL DEFAULT 'competitive',
    scoring_config    JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_by        UUID REFERENCES "user".users (id) ON DELETE SET NULL,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT quizgame_assignments_grade_policy_chk
        CHECK (grade_policy IN ('best', 'last', 'average')),
    CONSTRAINT quizgame_assignments_attempts_chk
        CHECK (attempts_allowed >= 1 AND attempts_allowed <= 100)
);

CREATE INDEX IF NOT EXISTS idx_quizgame_assignments_course
    ON quizgame.assignments (course_id, due_at);

COMMENT ON TABLE quizgame.assignments IS
    'IQ.6: Async homework binding a kit to a course with open/due/close windows.';

CREATE TABLE IF NOT EXISTS quizgame.assignment_attempts (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    assignment_id UUID NOT NULL REFERENCES quizgame.assignments (id) ON DELETE CASCADE,
    user_id       UUID NOT NULL REFERENCES "user".users (id) ON DELETE CASCADE,
    session_id    UUID NOT NULL REFERENCES quizgame.sessions (id) ON DELETE CASCADE,
    attempt_no    INTEGER NOT NULL,
    score         INTEGER NOT NULL DEFAULT 0,
    submitted_at  TIMESTAMPTZ,
    is_late       BOOLEAN NOT NULL DEFAULT FALSE,
    UNIQUE (assignment_id, user_id, attempt_no)
);

CREATE INDEX IF NOT EXISTS idx_quizgame_assignment_attempts_user
    ON quizgame.assignment_attempts (assignment_id, user_id);

COMMENT ON TABLE quizgame.assignment_attempts IS
    'IQ.6: Per-student homework runs; each attempt owns a mode=homework session.';

-- Effective gradebook scores (policy-applied); IQ.7 syncs these into coursegrades.
CREATE TABLE IF NOT EXISTS quizgame.assignment_grades (
    assignment_id UUID NOT NULL REFERENCES quizgame.assignments (id) ON DELETE CASCADE,
    user_id       UUID NOT NULL REFERENCES "user".users (id) ON DELETE CASCADE,
    score         NUMERIC(8, 2) NOT NULL,
    policy        TEXT NOT NULL,
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (assignment_id, user_id)
);

COMMENT ON TABLE quizgame.assignment_grades IS
    'IQ.6: Policy-applied homework grades (best/last/average) pending IQ.7 gradebook sync.';

-- Mode sub-flags (plan §15): ship team → student-paced → homework independently.
ALTER TABLE settings.platform_app_settings
    ADD COLUMN IF NOT EXISTS ff_iq_team_mode BOOLEAN,
    ADD COLUMN IF NOT EXISTS ff_iq_student_paced BOOLEAN,
    ADD COLUMN IF NOT EXISTS ff_iq_homework BOOLEAN;

COMMENT ON COLUMN settings.platform_app_settings.ff_iq_team_mode IS
    'IQ.6: Enable team game mode. Requires ff_interactive_quizzes + ff_iq_live_hosting.';
COMMENT ON COLUMN settings.platform_app_settings.ff_iq_student_paced IS
    'IQ.6: Enable student-paced game mode. Requires ff_interactive_quizzes + ff_iq_live_hosting.';
COMMENT ON COLUMN settings.platform_app_settings.ff_iq_homework IS
    'IQ.6: Enable async homework assignments. Requires ff_interactive_quizzes.';
