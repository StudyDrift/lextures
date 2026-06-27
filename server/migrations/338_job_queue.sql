-- Generic background job queue (plan 17.3). A single durable Postgres-backed
-- queue that any job type can enqueue onto. Workers claim rows with
-- SELECT ... FOR UPDATE SKIP LOCKED so multiple app instances process jobs
-- without double-execution, and a visibility timeout reclaims rows whose
-- worker crashed mid-run.

CREATE SCHEMA IF NOT EXISTS jobs;

CREATE TABLE jobs.queue (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    job_type      TEXT NOT NULL,
    payload       JSONB NOT NULL DEFAULT '{}'::jsonb,
    priority      SMALLINT NOT NULL DEFAULT 5,   -- 1 = highest, 10 = lowest
    status        TEXT NOT NULL DEFAULT 'pending', -- pending | running | completed | failed
    attempts      SMALLINT NOT NULL DEFAULT 0,
    max_attempts  SMALLINT NOT NULL DEFAULT 5,
    unique_key    TEXT,
    scheduled_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    started_at    TIMESTAMPTZ,
    completed_at  TIMESTAMPTZ,
    error_log     TEXT,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Deduplication: a logical job (identified by unique_key) may only have one
-- in-flight copy queued at a time. Once it completes or dead-letters the key is
-- free to be enqueued again.
CREATE UNIQUE INDEX uniq_jobs_queue_unique_key
    ON jobs.queue (unique_key)
    WHERE unique_key IS NOT NULL AND status IN ('pending', 'running', 'failed');

-- Claim path: workers scan ready rows ordered by priority then schedule.
CREATE INDEX idx_jobs_queue_claim
    ON jobs.queue (priority, scheduled_at)
    WHERE status IN ('pending', 'failed');

-- Visibility-timeout reclaim path: find running rows that started too long ago.
CREATE INDEX idx_jobs_queue_running_started
    ON jobs.queue (started_at)
    WHERE status = 'running';

CREATE TABLE jobs.dead_letters (
    id          UUID PRIMARY KEY,
    job_type    TEXT NOT NULL,
    payload     JSONB NOT NULL,
    priority    SMALLINT NOT NULL DEFAULT 5,
    unique_key  TEXT,
    attempts    SMALLINT NOT NULL DEFAULT 0,
    error_log   TEXT,
    failed_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    redriven    BOOLEAN NOT NULL DEFAULT false
);

CREATE INDEX idx_jobs_dead_letters_failed_at
    ON jobs.dead_letters (failed_at DESC);

COMMENT ON TABLE jobs.queue IS
    'Generic durable background job queue (plan 17.3). Claimed via SELECT ... FOR UPDATE SKIP LOCKED.';
COMMENT ON TABLE jobs.dead_letters IS
    'Jobs that exhausted max_attempts; re-drivable from the admin jobs UI.';
