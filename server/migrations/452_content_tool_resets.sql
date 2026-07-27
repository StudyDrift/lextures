-- CT.4 — Content Tools instructor reset snapshots, async jobs, and org retention.

-- Snapshot of a learner's tool state immediately before a reset. Restorable within retention.
CREATE TABLE IF NOT EXISTS course.content_tool_state_resets (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    instance_id      UUID NOT NULL REFERENCES course.content_tool_instances (id) ON DELETE CASCADE,
    enrollment_id    UUID NOT NULL REFERENCES course.course_enrollments (id) ON DELETE CASCADE,
    course_id        UUID NOT NULL REFERENCES course.courses (id) ON DELETE CASCADE,
    tool_id          TEXT NOT NULL,
    scope            TEXT NOT NULL CHECK (scope IN
                       ('instance_enrollment','instance_all','item_enrollment','item_all',
                        'course_enrollment','self')),
    reason           TEXT,
    prior_state_json JSONB NOT NULL,
    prior_status     TEXT NOT NULL,
    prior_score_raw  NUMERIC(10,4),
    prior_score_max  NUMERIC(10,4),
    prior_revision   BIGINT NOT NULL,
    batch_id         UUID,                    -- groups one bulk operation
    reset_by         UUID REFERENCES "user".users (id) ON DELETE SET NULL,
    reset_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    restored_at      TIMESTAMPTZ,
    restored_by      UUID REFERENCES "user".users (id) ON DELETE SET NULL,
    purge_after      TIMESTAMPTZ NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_ctsr_instance_enrollment
    ON course.content_tool_state_resets (instance_id, enrollment_id, reset_at DESC);
CREATE INDEX IF NOT EXISTS idx_ctsr_batch ON course.content_tool_state_resets (batch_id);
CREATE INDEX IF NOT EXISTS idx_ctsr_purge ON course.content_tool_state_resets (purge_after);
CREATE INDEX IF NOT EXISTS idx_ctsr_course_created
    ON course.content_tool_state_resets (course_id, reset_at DESC);

COMMENT ON TABLE course.content_tool_state_resets IS
    'CT.4: Snapshot of content tool state before reset; restorable until purge_after.';

-- Async bulk-reset jobs (mirrors the shipped job-record pattern).
CREATE TABLE IF NOT EXISTS course.content_tool_reset_jobs (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    course_id      UUID NOT NULL REFERENCES course.courses (id) ON DELETE CASCADE,
    requested_by   UUID REFERENCES "user".users (id) ON DELETE SET NULL,
    scope          TEXT NOT NULL,
    target_json    JSONB NOT NULL,            -- {instanceId?, itemId?, enrollmentId?, sectionIds?}
    reason         TEXT,
    notify         BOOLEAN NOT NULL DEFAULT TRUE,
    status         TEXT NOT NULL DEFAULT 'queued'
                     CHECK (status IN ('queued','running','succeeded','failed','cancelled')),
    total_rows     INTEGER NOT NULL DEFAULT 0,
    processed_rows INTEGER NOT NULL DEFAULT 0,
    batch_id       UUID,
    error          TEXT,
    result_json    JSONB,
    idempotency_key TEXT UNIQUE,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    finished_at    TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS idx_ctrj_course_created ON course.content_tool_reset_jobs (course_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_ctrj_status ON course.content_tool_reset_jobs (status) WHERE status IN ('queued','running');

COMMENT ON TABLE course.content_tool_reset_jobs IS
    'CT.4: Async bulk Content Tool reset jobs (FR-13).';

-- Org-level retention for snapshots (default 90 days).
ALTER TABLE tenant.organizations
    ADD COLUMN IF NOT EXISTS content_tool_state_retention_days INTEGER NOT NULL DEFAULT 90;
COMMENT ON COLUMN tenant.organizations.content_tool_state_retention_days IS
    'Days a Content Tools reset snapshot remains restorable before nightly purge (plan CT.4 FR-9).';
