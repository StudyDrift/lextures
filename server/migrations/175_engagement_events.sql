-- 9.7 Engagement Metrics: events table, summaries cache, and feature flag.
-- Depends on: analytics schema (171), user.users (001), course.course_enrollments, course.courses.

CREATE SCHEMA IF NOT EXISTS analytics;

-- High-volume append-only event log, partitioned by month.
CREATE TABLE analytics.engagement_events (
    id          BIGSERIAL,
    user_id     UUID NOT NULL REFERENCES "user".users (id) ON DELETE CASCADE,
    course_id   UUID REFERENCES course.courses (id) ON DELETE CASCADE,
    item_id     UUID,
    item_type   TEXT,
    event_type  TEXT NOT NULL,
    value       REAL,
    occurred_at TIMESTAMPTZ NOT NULL DEFAULT now()
) PARTITION BY RANGE (occurred_at);

CREATE TABLE analytics.engagement_events_default
    PARTITION OF analytics.engagement_events DEFAULT;

CREATE TABLE analytics.engagement_events_2026_05
    PARTITION OF analytics.engagement_events
    FOR VALUES FROM ('2026-05-01') TO ('2026-06-01');

CREATE TABLE analytics.engagement_events_2026_06
    PARTITION OF analytics.engagement_events
    FOR VALUES FROM ('2026-06-01') TO ('2026-07-01');

CREATE TABLE analytics.engagement_events_2026_07
    PARTITION OF analytics.engagement_events
    FOR VALUES FROM ('2026-07-01') TO ('2026-08-01');

CREATE INDEX idx_engagement_events_user_occurred
    ON analytics.engagement_events (user_id, occurred_at DESC);

CREATE INDEX idx_engagement_events_course_item_type
    ON analytics.engagement_events (course_id, item_id, event_type);

-- Pre-aggregated daily summaries per enrollment (populated by nightly job or on-demand).
CREATE TABLE analytics.engagement_summaries (
    enrollment_id        UUID NOT NULL REFERENCES course.course_enrollments (id) ON DELETE CASCADE,
    summary_date         DATE NOT NULL,
    total_time_on_task_s INTEGER NOT NULL DEFAULT 0,
    logins_last_7_days   INTEGER NOT NULL DEFAULT 0,
    avg_video_watch_pct  REAL,
    avg_scroll_depth     REAL,
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (enrollment_id, summary_date)
);

CREATE INDEX idx_engagement_summaries_enrollment
    ON analytics.engagement_summaries (enrollment_id, summary_date DESC);

-- Feature flag.
ALTER TABLE settings.platform_app_settings
    ADD COLUMN IF NOT EXISTS engagement_tracking_enabled BOOLEAN;
