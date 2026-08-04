-- CC.2: Course checklist dismissals, evaluation snapshots, and audit events.

CREATE TABLE IF NOT EXISTS course.course_checklist_item_state (
    course_id            UUID NOT NULL REFERENCES course.courses (id) ON DELETE CASCADE,
    item_id              TEXT NOT NULL,
    dismissed_at         TIMESTAMPTZ,
    dismissed_by_user_id UUID REFERENCES "user".users (id) ON DELETE SET NULL,
    dismiss_reason       TEXT NOT NULL DEFAULT '',
    dismiss_note         TEXT NOT NULL DEFAULT '',
    snoozed_until        TIMESTAMPTZ,
    restored_at          TIMESTAMPTZ,
    restored_by_user_id  UUID REFERENCES "user".users (id) ON DELETE SET NULL,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (course_id, item_id),
    CONSTRAINT course_checklist_item_id_format CHECK (item_id ~ '^[a-z][a-z0-9]*(\.[a-z0-9-]+){1,3}$'),
    CONSTRAINT course_checklist_reason_check CHECK (
        dismiss_reason IN ('', 'not_applicable', 'done_elsewhere', 'disagree', 'later', 'other')),
    CONSTRAINT course_checklist_note_len CHECK (length(dismiss_note) <= 500)
);

CREATE INDEX IF NOT EXISTS idx_course_checklist_state_dismissed
    ON course.course_checklist_item_state (course_id) WHERE dismissed_at IS NOT NULL;

CREATE TABLE IF NOT EXISTS course.course_checklist_snapshots (
    course_id             UUID PRIMARY KEY REFERENCES course.courses (id) ON DELETE CASCADE,
    computed_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    engine_version        INT NOT NULL,
    catalog_version       TEXT NOT NULL,
    payload               JSONB NOT NULL,
    total_count           INT NOT NULL DEFAULT 0,
    done_count            INT NOT NULL DEFAULT 0,
    outstanding_essential INT NOT NULL DEFAULT 0,
    outstanding_total     INT NOT NULL DEFAULT 0,
    dismissed_count       INT NOT NULL DEFAULT 0,
    CONSTRAINT course_checklist_payload_size CHECK (pg_column_size(payload) <= 262144)
);

CREATE INDEX IF NOT EXISTS idx_course_checklist_snapshots_computed
    ON course.course_checklist_snapshots (computed_at);

CREATE TABLE IF NOT EXISTS course.course_checklist_events (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    course_id     UUID NOT NULL REFERENCES course.courses (id) ON DELETE CASCADE,
    item_id       TEXT NOT NULL,
    action        TEXT NOT NULL CHECK (action IN ('dismiss', 'restore', 'complete', 'regress')),
    actor_user_id UUID REFERENCES "user".users (id) ON DELETE SET NULL,
    reason        TEXT NOT NULL DEFAULT '',
    occurred_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_course_checklist_events_course_time
    ON course.course_checklist_events (course_id, occurred_at DESC);

COMMENT ON TABLE course.course_checklist_item_state IS
    'CC.2: Per-course checklist item dismissals (course-scoped, not per-user).';
COMMENT ON TABLE course.course_checklist_snapshots IS
    'CC.2: Cached checklist evaluation Result plus denormalised badge counters.';
COMMENT ON TABLE course.course_checklist_events IS
    'CC.2: Audit trail for checklist dismiss/restore/status transitions.';
