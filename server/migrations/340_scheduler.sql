-- Scheduler / cron (plan 17.4). A thin layer over the generic background job
-- queue (17.3): a tick loop on every instance evaluates a configuration-driven
-- list of cron schedules and, guarded by a Postgres-backed distributed lock,
-- enqueues a regular jobs.queue row when a schedule is due. Actual execution is
-- therefore subject to the same retry / dead-letter logic as any other job.

-- One row per scheduled trigger: which schedule fired, when, and the queue job
-- it enqueued. Powers the admin schedule-history page (plan 17.4 FR-5, AC-4).
CREATE TABLE jobs.schedule_history (
    id           BIGSERIAL PRIMARY KEY,
    job_name     TEXT NOT NULL,
    triggered_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    job_id       UUID REFERENCES jobs.queue(id) ON DELETE SET NULL,
    status       TEXT NOT NULL DEFAULT 'triggered',
    notes        TEXT
);

CREATE INDEX idx_schedule_history_name_time
    ON jobs.schedule_history (job_name, triggered_at DESC);

-- Distributed lock: an instance must hold the lock for a job_name to fire it,
-- so only one instance enqueues per trigger even when many run (plan 17.4 FR-2,
-- AC-2). The lock is acquired only when no unexpired holder exists; expires_at
-- gives a TTL shorter than the schedule interval so a crashed holder cannot
-- starve the schedule (plan 17.4 NFR reliability).
CREATE TABLE jobs.schedule_locks (
    job_name   TEXT PRIMARY KEY,
    locked_by  TEXT NOT NULL,
    locked_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at TIMESTAMPTZ NOT NULL
);

-- Persisted enable/disable state so an admin can pause a schedule without a code
-- deploy (plan 17.4 FR-6). Absence of a row means "use the code default".
CREATE TABLE jobs.schedule_overrides (
    job_name   TEXT PRIMARY KEY,
    enabled    BOOLEAN NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

COMMENT ON TABLE jobs.schedule_history IS
    'One row per scheduled-job trigger (plan 17.4). job_id links to the enqueued jobs.queue row.';
COMMENT ON TABLE jobs.schedule_locks IS
    'Distributed lock so exactly one instance fires a given schedule per trigger (plan 17.4 FR-2).';
COMMENT ON TABLE jobs.schedule_overrides IS
    'Admin enable/disable overrides for scheduled jobs, applied without a deploy (plan 17.4 FR-6).';

-- Late-submission marker set by the late_submission_sweep scheduled job
-- (plan 17.4 FR-4, AC-1). A submission is late when it was submitted after the
-- assignment due date; the sweep keeps this accurate without instructor action.
ALTER TABLE course.module_assignment_submissions
    ADD COLUMN is_late BOOLEAN NOT NULL DEFAULT false;

COMMENT ON COLUMN course.module_assignment_submissions.is_late IS
    'True when submitted after the assignment due date; maintained by the late_submission_sweep scheduled job (plan 17.4).';
