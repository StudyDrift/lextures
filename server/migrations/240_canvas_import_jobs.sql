-- Queued Canvas LMS course imports (RabbitMQ worker + progress WebSocket).

CREATE SCHEMA IF NOT EXISTS jobs;

CREATE TABLE IF NOT EXISTS jobs.canvas_import_jobs (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id          UUID NOT NULL REFERENCES "user".users (id) ON DELETE CASCADE,
    course_code      TEXT NOT NULL,
    status           TEXT NOT NULL DEFAULT 'queued'
        CHECK (status IN ('queued', 'processing', 'completed', 'failed')),
    mode             TEXT NOT NULL,
    canvas_base_url  TEXT NOT NULL,
    canvas_course_id TEXT NOT NULL,
    include          JSONB NOT NULL DEFAULT '{}'::jsonb,
    last_progress    TEXT,
    error_message    TEXT,
    course_title     TEXT,
    attempts         SMALLINT NOT NULL DEFAULT 0,
    max_attempts     SMALLINT NOT NULL DEFAULT 3,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    started_at       TIMESTAMPTZ,
    completed_at     TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS canvas_import_jobs_status_created_idx
    ON jobs.canvas_import_jobs (status, created_at)
    WHERE status IN ('queued', 'processing');

CREATE INDEX IF NOT EXISTS canvas_import_jobs_user_created_idx
    ON jobs.canvas_import_jobs (user_id, created_at DESC);
