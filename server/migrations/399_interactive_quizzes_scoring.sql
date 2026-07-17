-- IQ.5 — Scoring profiles, explainable breakdowns, leaderboard privacy, power-up ledger.

ALTER TABLE quizgame.sessions
    ADD COLUMN IF NOT EXISTS scoring_profile     TEXT NOT NULL DEFAULT 'competitive',
    ADD COLUMN IF NOT EXISTS scoring_profile_ver INTEGER NOT NULL DEFAULT 1,
    ADD COLUMN IF NOT EXISTS scoring_config      JSONB NOT NULL DEFAULT '{}'::jsonb,
    ADD COLUMN IF NOT EXISTS leaderboard_privacy TEXT NOT NULL DEFAULT 'names';

COMMENT ON COLUMN quizgame.sessions.scoring_profile IS
    'IQ.5: competitive | formative | custom — fixed at game start.';
COMMENT ON COLUMN quizgame.sessions.scoring_profile_ver IS
    'IQ.5: scoring function version for historical reproducibility.';
COMMENT ON COLUMN quizgame.sessions.scoring_config IS
    'IQ.5: base, speedWeight, streakStep, streakCap, powerUpsEnabled, participationPoints.';
COMMENT ON COLUMN quizgame.sessions.leaderboard_privacy IS
    'IQ.5: names | nicknames | hidden — projector/public leaderboard display mode.';

ALTER TABLE quizgame.session_responses
    ADD COLUMN IF NOT EXISTS points_breakdown JSONB NOT NULL DEFAULT '{}'::jsonb;

COMMENT ON COLUMN quizgame.session_responses.points_breakdown IS
    'IQ.5: Explainable award {base, speedBonus, streakBonus, styleMultiplier, powerUp, total}.';

CREATE TABLE IF NOT EXISTS quizgame.player_powerups (
    session_id     UUID NOT NULL REFERENCES quizgame.sessions (id) ON DELETE CASCADE,
    player_id      UUID NOT NULL REFERENCES quizgame.session_players (id) ON DELETE CASCADE,
    question_index INTEGER NOT NULL,
    kind           TEXT NOT NULL,
    applied_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (session_id, player_id, question_index, kind)
);

CREATE INDEX IF NOT EXISTS idx_quizgame_player_powerups_session
    ON quizgame.player_powerups (session_id, player_id);

COMMENT ON TABLE quizgame.player_powerups IS
    'IQ.5: Server-adjudicated power-up ledger (double_or_nothing | shield).';
