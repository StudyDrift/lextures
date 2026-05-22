-- Report PDF export feature (plan 9.8).
-- Adds report_schedules for recurring delivery and the report_export_enabled platform flag.

ALTER TABLE settings.platform_app_settings
    ADD COLUMN IF NOT EXISTS report_export_enabled BOOLEAN;

CREATE SCHEMA IF NOT EXISTS analytics;

CREATE TABLE IF NOT EXISTS analytics.report_schedules (
  id             UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
  owner_id       UUID        NOT NULL REFERENCES "user".users(id) ON DELETE CASCADE,
  course_id      UUID        REFERENCES course.courses(id) ON DELETE CASCADE,
  report_type    TEXT        NOT NULL,
  parameters     JSONB       NOT NULL DEFAULT '{}',
  recipients     TEXT[]      NOT NULL,
  cadence        TEXT        NOT NULL,
  cadence_detail JSONB,
  enabled        BOOLEAN     NOT NULL DEFAULT true,
  last_run_at    TIMESTAMPTZ,
  next_run_at    TIMESTAMPTZ NOT NULL,
  created_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS report_schedules_next_run_idx
    ON analytics.report_schedules (next_run_at)
    WHERE enabled = true;

CREATE INDEX IF NOT EXISTS report_schedules_owner_idx
    ON analytics.report_schedules (owner_id);
