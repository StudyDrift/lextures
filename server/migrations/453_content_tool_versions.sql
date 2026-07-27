-- CT.5 — Content Tools version registry mirror, migration quarantine, and eager migration jobs.

-- Registry mirror: one row per (tool_id, version) actually seen by this deployment.
CREATE TABLE IF NOT EXISTS course.content_tool_versions (
    tool_id               TEXT NOT NULL,
    version               TEXT NOT NULL,
    manifest_json         JSONB NOT NULL,
    config_schema_version INTEGER NOT NULL DEFAULT 1,
    state_schema_version  INTEGER NOT NULL DEFAULT 1,
    sandbox_mode          TEXT NOT NULL DEFAULT 'inprocess'
                            CHECK (sandbox_mode IN ('inprocess','iframe')),
    status                TEXT NOT NULL DEFAULT 'active'
                            CHECK (status IN ('active','deprecated','sunset','disabled')),
    breaker_open_at       TIMESTAMPTZ,
    sunset_at             TIMESTAMPTZ,
    first_seen_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (tool_id, version)
);
CREATE INDEX IF NOT EXISTS idx_ctv_tool_status ON course.content_tool_versions (tool_id, status);

COMMENT ON TABLE course.content_tool_versions IS
    'CT.5: Mirror of registered Content Tool versions with status and circuit-breaker state.';

-- Documents that failed migration; the original is preserved verbatim.
CREATE TABLE IF NOT EXISTS course.content_tool_state_quarantine (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    state_id       UUID NOT NULL REFERENCES course.content_tool_states (id) ON DELETE CASCADE,
    tool_id        TEXT NOT NULL,
    from_version   INTEGER NOT NULL,
    to_version     INTEGER NOT NULL,
    error          TEXT NOT NULL,
    original_json  JSONB NOT NULL,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    resolved_at    TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS idx_ctsq_tool ON course.content_tool_state_quarantine (tool_id, resolved_at);

COMMENT ON TABLE course.content_tool_state_quarantine IS
    'CT.5: Original state documents that failed schema migration; never dropped.';

-- Eager migration jobs.
CREATE TABLE IF NOT EXISTS course.content_tool_migration_jobs (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tool_id         TEXT NOT NULL,
    from_version    INTEGER NOT NULL,
    to_version      INTEGER NOT NULL,
    dry_run         BOOLEAN NOT NULL DEFAULT TRUE,
    status          TEXT NOT NULL DEFAULT 'queued'
                      CHECK (status IN ('queued','running','succeeded','failed','cancelled')),
    total_docs      INTEGER NOT NULL DEFAULT 0,
    migrated_docs   INTEGER NOT NULL DEFAULT 0,
    failed_docs     INTEGER NOT NULL DEFAULT 0,
    cursor_state_id UUID,
    error           TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    finished_at     TIMESTAMPTZ
);

COMMENT ON TABLE course.content_tool_migration_jobs IS
    'CT.5: Eager (or dry-run) state migration jobs with resumable cursor.';
