-- CT.3: Content Tools student runtime — action idempotency + state lookup index.

CREATE TABLE IF NOT EXISTS course.content_tool_action_idempotency (
    idempotency_key TEXT PRIMARY KEY,
    instance_id     UUID NOT NULL REFERENCES course.content_tool_instances (id) ON DELETE CASCADE,
    enrollment_id   UUID NOT NULL REFERENCES course.course_enrollments (id) ON DELETE CASCADE,
    action          TEXT NOT NULL,
    result_json     JSONB NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_ctai_created
    ON course.content_tool_action_idempotency (created_at);

-- Fast "what has this learner touched" lookups for the reader's batched load.
CREATE INDEX IF NOT EXISTS idx_cts_enrollment_instance
    ON course.content_tool_states (enrollment_id, instance_id) WHERE scope = 'enrollment';
