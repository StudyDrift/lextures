-- Rollback IQ.6 game modes.

ALTER TABLE settings.platform_app_settings
    DROP COLUMN IF EXISTS ff_iq_homework,
    DROP COLUMN IF EXISTS ff_iq_student_paced,
    DROP COLUMN IF EXISTS ff_iq_team_mode;

DROP TABLE IF EXISTS quizgame.assignment_grades;
DROP TABLE IF EXISTS quizgame.assignment_attempts;
DROP TABLE IF EXISTS quizgame.assignments;

ALTER TABLE quizgame.session_players
    DROP COLUMN IF EXISTS finished_at,
    DROP COLUMN IF EXISTS time_budget_ends_at,
    DROP COLUMN IF EXISTS question_order,
    DROP COLUMN IF EXISTS question_deadline_at,
    DROP COLUMN IF EXISTS question_opened_at,
    DROP COLUMN IF EXISTS current_phase,
    DROP COLUMN IF EXISTS current_index;

ALTER TABLE quizgame.session_players
    DROP CONSTRAINT IF EXISTS session_players_team_id_fkey;

DROP TABLE IF EXISTS quizgame.teams;
