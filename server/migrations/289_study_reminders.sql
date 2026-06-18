-- Plan 15.10 — Daily study goals and reminder scheduling for self-learners.

CREATE SCHEMA IF NOT EXISTS studyreminders;

CREATE TABLE studyreminders.configs (
    user_id             UUID PRIMARY KEY REFERENCES "user".users (id) ON DELETE CASCADE,
    daily_goal_minutes  INT NOT NULL DEFAULT 20,
    reminder_time       TIME NOT NULL DEFAULT '19:00',
    reminder_channels   TEXT[] NOT NULL DEFAULT '{"email"}',
    paused_until        DATE,
    weekly_summary      BOOLEAN NOT NULL DEFAULT TRUE,
    enabled             BOOLEAN NOT NULL DEFAULT FALSE,
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE studyreminders.configs IS
    'Per-user daily study goal and reminder preferences (plan 15.10).';

CREATE TABLE studyreminders.send_log (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL REFERENCES "user".users (id) ON DELETE CASCADE,
    send_date       DATE NOT NULL,
    reminder_type   TEXT NOT NULL,
    channel         TEXT NOT NULL,
    idempotency_key TEXT UNIQUE,
    sent_at         TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE studyreminders.send_log IS
    'At-most-once reminder delivery log with per-day deduplication (plan 15.10).';

CREATE UNIQUE INDEX idx_studyreminders_send_log_dedup
    ON studyreminders.send_log (user_id, send_date, reminder_type, channel);

ALTER TABLE settings.platform_app_settings
    ADD COLUMN IF NOT EXISTS ff_study_reminders BOOLEAN;

COMMENT ON COLUMN settings.platform_app_settings.ff_study_reminders IS
    'Enables daily study goal reminders and weekly progress summaries (plan 15.10).';
