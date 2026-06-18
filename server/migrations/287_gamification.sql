-- Plan 15.9 — Streaks, XP, leaderboards, and milestone badges for self-learners.

CREATE SCHEMA IF NOT EXISTS gamification;

CREATE TABLE gamification.user_gamification (
    user_id              UUID PRIMARY KEY REFERENCES "user".users (id) ON DELETE CASCADE,
    xp_total             INT NOT NULL DEFAULT 0,
    current_streak       INT NOT NULL DEFAULT 0,
    longest_streak       INT NOT NULL DEFAULT 0,
    last_activity_date   DATE,
    streak_freezes       INT NOT NULL DEFAULT 0,
    freeze_cover_date    DATE,
    leaderboard_visible  BOOLEAN NOT NULL DEFAULT TRUE,
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE gamification.user_gamification IS
    'Per-user gamification aggregate: XP balance, streak counters, and freeze inventory (plan 15.9).';

CREATE TABLE gamification.xp_events (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL REFERENCES "user".users (id) ON DELETE CASCADE,
    course_id       UUID REFERENCES course.courses (id) ON DELETE SET NULL,
    activity_type   TEXT NOT NULL,
    source_id       UUID,
    xp_awarded      INT NOT NULL,
    idempotency_key TEXT NOT NULL UNIQUE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE gamification.xp_events IS
    'Append-only XP ledger; idempotency_key prevents duplicate awards (plan 15.9).';

CREATE INDEX idx_xp_events_user ON gamification.xp_events (user_id, created_at DESC);
CREATE INDEX idx_xp_events_course ON gamification.xp_events (course_id, user_id);

CREATE TABLE gamification.user_badges (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID NOT NULL REFERENCES "user".users (id) ON DELETE CASCADE,
    badge_type TEXT NOT NULL,
    awarded_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (user_id, badge_type)
);

COMMENT ON TABLE gamification.user_badges IS
    'Milestone badges awarded once per user (plan 15.9).';

CREATE INDEX idx_user_badges_user ON gamification.user_badges (user_id, awarded_at DESC);

ALTER TABLE course.courses
    ADD COLUMN IF NOT EXISTS gamification_enabled BOOLEAN NOT NULL DEFAULT FALSE;

COMMENT ON COLUMN course.courses.gamification_enabled IS
    'When true, learners earn XP and appear on the course leaderboard (plan 15.9).';

UPDATE course.courses
SET gamification_enabled = TRUE
WHERE course_mode = 'self_paced';

ALTER TABLE settings.platform_app_settings
    ADD COLUMN IF NOT EXISTS ff_gamification BOOLEAN;

COMMENT ON COLUMN settings.platform_app_settings.ff_gamification IS
    'Enables streaks, XP, leaderboards, and badges for self-learner courses (plan 15.9).';
