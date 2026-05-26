-- 9.10 Instructor "What's Working" Signals: weekly insight aggregation per course.
-- Depends on: analytics schema (175), course.courses (001), "user".users.

CREATE TABLE analytics.instructor_insights (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    course_id    UUID NOT NULL REFERENCES course.courses (id) ON DELETE CASCADE,
    week_of      DATE NOT NULL,
    working_well    JSONB NOT NULL DEFAULT '[]',
    needs_attention JSONB NOT NULL DEFAULT '[]',
    scatter_data    JSONB NOT NULL DEFAULT '[]',
    generated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (course_id, week_of)
);

CREATE INDEX idx_instructor_insights_course
    ON analytics.instructor_insights (course_id, week_of DESC);

CREATE TABLE analytics.dismissed_signals (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    course_id    UUID NOT NULL REFERENCES course.courses (id) ON DELETE CASCADE,
    signal_key   TEXT NOT NULL,
    dismissed_by UUID NOT NULL REFERENCES "user".users (id) ON DELETE CASCADE,
    reason       TEXT,
    dismissed_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (course_id, signal_key)
);

CREATE INDEX idx_dismissed_signals_course
    ON analytics.dismissed_signals (course_id);

-- Feature flag.
ALTER TABLE settings.platform_app_settings
    ADD COLUMN IF NOT EXISTS instructor_insights_enabled BOOLEAN;
