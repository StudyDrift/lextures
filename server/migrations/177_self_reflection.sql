-- 9.9 Learner self-reflection & study-skills coaching dashboard.

CREATE TABLE analytics.study_goals (
    user_id       UUID PRIMARY KEY REFERENCES "user".users (id) ON DELETE CASCADE,
    weekly_hours  REAL NOT NULL DEFAULT 0,
    opted_in      BOOLEAN NOT NULL DEFAULT false,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE analytics.reflection_journal (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID NOT NULL REFERENCES "user".users (id) ON DELETE CASCADE,
    course_id   UUID REFERENCES course.courses (id) ON DELETE SET NULL,
    entry_text  TEXT NOT NULL CHECK (char_length(entry_text) <= 280),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_reflection_journal_user_created
    ON analytics.reflection_journal (user_id, created_at DESC);

CREATE TABLE analytics.coaching_tips (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id      UUID NOT NULL REFERENCES "user".users (id) ON DELETE CASCADE,
    tip_text     TEXT NOT NULL,
    week_of      DATE NOT NULL,
    delivered_at TIMESTAMPTZ,
    rating       SMALLINT CHECK (rating IS NULL OR rating IN (-1, 1)),
    UNIQUE (user_id, week_of)
);

CREATE INDEX idx_coaching_tips_user_week
    ON analytics.coaching_tips (user_id, week_of DESC);

ALTER TABLE settings.platform_app_settings
    ADD COLUMN IF NOT EXISTS self_reflection_enabled BOOLEAN;
