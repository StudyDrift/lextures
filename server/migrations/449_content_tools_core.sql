-- CT.1 — Content Tools foundations: per-course flag, settings, instances, states, events.

-- Per-course flag (mirrors adaptive_content_enabled / adaptive_paths_enabled).
ALTER TABLE course.courses
    ADD COLUMN IF NOT EXISTS content_tools_enabled BOOLEAN NOT NULL DEFAULT FALSE;
COMMENT ON COLUMN course.courses.content_tools_enabled IS
    'When true, interactive Content Tools may be inserted into section bodies and may store per-enrollment state (plan CT.1).';

-- Per-course configuration, created lazily on first PUT.
CREATE TABLE IF NOT EXISTS course.content_tool_settings (
    course_id             UUID PRIMARY KEY REFERENCES course.courses (id) ON DELETE CASCADE,
    allowed_tool_ids      TEXT[] NOT NULL DEFAULT '{}',   -- empty = all org-permitted tools
    student_reset_allowed BOOLEAN NOT NULL DEFAULT FALSE, -- may a student clear their own state?
    max_instances_per_item SMALLINT NOT NULL DEFAULT 50 CHECK (max_instances_per_item BETWEEN 1 AND 200),
    updated_by            UUID REFERENCES "user".users (id) ON DELETE SET NULL,
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE course.content_tool_settings IS
    'CT.1: Per-course Content Tools configuration. Created lazily on first settings PUT.';

-- One placed tool. The body Markdown carries only id; everything else lives here.
CREATE TABLE IF NOT EXISTS course.content_tool_instances (
    id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    course_id             UUID NOT NULL REFERENCES course.courses (id) ON DELETE CASCADE,
    structure_item_id     UUID REFERENCES course.course_structure_items (id) ON DELETE CASCADE,
    host_kind             TEXT NOT NULL CHECK (host_kind IN
                            ('content_page','assignment','quiz','syllabus','portfolio_artifact')),
    section_key           TEXT,            -- editor section this instance sits in (advisory only)
    tool_id               TEXT NOT NULL,
    tool_version          TEXT NOT NULL,   -- semver pinned at insert (CT.5 uses it to migrate)
    title                 TEXT,
    config_json           JSONB NOT NULL DEFAULT '{}'::jsonb,
    config_schema_version INTEGER NOT NULL DEFAULT 1,
    status                TEXT NOT NULL DEFAULT 'active'
                            CHECK (status IN ('active','archived')),
    created_by            UUID REFERENCES "user".users (id) ON DELETE SET NULL,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT content_tool_instances_config_size CHECK (pg_column_size(config_json) <= 262144),
    CONSTRAINT content_tool_instances_syllabus_shape CHECK (
        (host_kind = 'syllabus' AND structure_item_id IS NULL)
        OR (host_kind <> 'syllabus' AND structure_item_id IS NOT NULL)
    )
);
CREATE INDEX IF NOT EXISTS idx_cti_item ON course.content_tool_instances (structure_item_id, status);
CREATE INDEX IF NOT EXISTS idx_cti_course_tool ON course.content_tool_instances (course_id, tool_id);

COMMENT ON TABLE course.content_tool_instances IS
    'CT.1: Placed Content Tool instance + config. Body Markdown stores only the instance id pointer.';

-- Per-enrollment learner state. One JSONB document per (instance, enrollment).
CREATE TABLE IF NOT EXISTS course.content_tool_states (
    id                   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    instance_id          UUID NOT NULL REFERENCES course.content_tool_instances (id) ON DELETE CASCADE,
    enrollment_id        UUID NOT NULL REFERENCES course.course_enrollments (id) ON DELETE CASCADE,
    user_id              UUID NOT NULL REFERENCES "user".users (id) ON DELETE CASCADE,
    state_json           JSONB NOT NULL DEFAULT '{}'::jsonb,
    state_schema_version INTEGER NOT NULL DEFAULT 1,
    revision             BIGINT NOT NULL DEFAULT 0,       -- optimistic concurrency token
    status               TEXT NOT NULL DEFAULT 'not_started'
                           CHECK (status IN ('not_started','in_progress','submitted','completed')),
    score_raw            NUMERIC(10,4),
    score_max            NUMERIC(10,4),
    interaction_count    INTEGER NOT NULL DEFAULT 0,
    first_interacted_at  TIMESTAMPTZ,
    last_interacted_at   TIMESTAMPTZ,
    completed_at         TIMESTAMPTZ,
    reset_count          INTEGER NOT NULL DEFAULT 0,
    last_reset_at        TIMESTAMPTZ,
    last_reset_by        UUID REFERENCES "user".users (id) ON DELETE SET NULL,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (instance_id, enrollment_id),
    CONSTRAINT content_tool_states_size CHECK (pg_column_size(state_json) <= 65536),
    CONSTRAINT content_tool_states_score_shape CHECK (
        (score_raw IS NULL AND score_max IS NULL) OR (score_max IS NOT NULL AND score_max > 0)
    )
);
CREATE INDEX IF NOT EXISTS idx_cts_enrollment ON course.content_tool_states (enrollment_id);
CREATE INDEX IF NOT EXISTS idx_cts_instance_status ON course.content_tool_states (instance_id, status);

COMMENT ON TABLE course.content_tool_states IS
    'CT.1: Per-enrollment Content Tool learner state (FERPA education record). Cascades with enrollment.';

-- Append-only log: authoring changes now, learner interactions from CT.3, resets from CT.4.
CREATE TABLE IF NOT EXISTS course.content_tool_events (
    id            BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    instance_id   UUID REFERENCES course.content_tool_instances (id) ON DELETE CASCADE,
    course_id     UUID NOT NULL REFERENCES course.courses (id) ON DELETE CASCADE,
    enrollment_id UUID REFERENCES course.course_enrollments (id) ON DELETE CASCADE,
    actor_user_id UUID REFERENCES "user".users (id) ON DELETE SET NULL,
    tool_id       TEXT NOT NULL,
    event_type    TEXT NOT NULL,   -- instance_created|instance_updated|instance_archived|…
    payload_json  JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_cte_instance_created ON course.content_tool_events (instance_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_cte_course_created   ON course.content_tool_events (course_id, created_at DESC);

COMMENT ON TABLE course.content_tool_events IS
    'CT.1: Append-only Content Tools audit/event log (authoring + future learner interactions).';
