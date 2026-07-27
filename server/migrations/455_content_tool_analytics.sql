-- CT.7 — Content Tools analytics: state summaries, daily rollups, gradebook links.

-- One row per learner state, maintained on write; the aggregation substrate.
CREATE TABLE IF NOT EXISTS analytics.content_tool_state_summaries (
    state_id       UUID PRIMARY KEY REFERENCES course.content_tool_states (id) ON DELETE CASCADE,
    instance_id    UUID NOT NULL REFERENCES course.content_tool_instances (id) ON DELETE CASCADE,
    course_id      UUID NOT NULL REFERENCES course.courses (id) ON DELETE CASCADE,
    enrollment_id  UUID NOT NULL REFERENCES course.course_enrollments (id) ON DELETE CASCADE,
    tool_id        TEXT NOT NULL,
    role           TEXT NOT NULL,            -- enrollment role at write time (filters staff rows)
    engaged        BOOLEAN NOT NULL DEFAULT FALSE,
    completed      BOOLEAN NOT NULL DEFAULT FALSE,
    score_pct      NUMERIC(5,2),
    duration_ms    INTEGER,
    facets_json    JSONB NOT NULL DEFAULT '{}'::jsonb,
    projection_version INTEGER NOT NULL DEFAULT 1,
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_ctss_instance ON analytics.content_tool_state_summaries (instance_id, role);
CREATE INDEX IF NOT EXISTS idx_ctss_course_tool ON analytics.content_tool_state_summaries (course_id, tool_id);
CREATE INDEX IF NOT EXISTS idx_ctss_facets ON analytics.content_tool_state_summaries USING GIN (facets_json jsonb_path_ops);
CREATE INDEX IF NOT EXISTS idx_ctss_enrollment ON analytics.content_tool_state_summaries (enrollment_id);

COMMENT ON TABLE analytics.content_tool_state_summaries IS
    'CT.7: Per-state typed summary projection for aggregation (never re-parse raw state_json at query time).';

-- Nightly cross-course rollup for platform/admin telemetry (no student content).
CREATE TABLE IF NOT EXISTS analytics.content_tool_daily_rollups (
    day            DATE NOT NULL,
    org_id         UUID REFERENCES tenant.organizations (id) ON DELETE CASCADE,
    tool_id        TEXT NOT NULL,
    instances      INTEGER NOT NULL DEFAULT 0,
    learners       INTEGER NOT NULL DEFAULT 0,
    engagements    INTEGER NOT NULL DEFAULT 0,
    completions    INTEGER NOT NULL DEFAULT 0,
    mean_score_pct NUMERIC(5,2),
    ai_tokens      BIGINT NOT NULL DEFAULT 0,
    ai_cost_usd    NUMERIC(12,4) NOT NULL DEFAULT 0,
    render_errors  INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (day, org_id, tool_id)
);

COMMENT ON TABLE analytics.content_tool_daily_rollups IS
    'CT.7: Nightly cross-course tool telemetry for platform admins (counts only, no free-text).';

-- Opt-in gradebook + outcome linkage, per instance.
CREATE TABLE IF NOT EXISTS course.content_tool_grade_links (
    instance_id    UUID PRIMARY KEY REFERENCES course.content_tool_instances (id) ON DELETE CASCADE,
    assignment_item_id UUID REFERENCES course.course_structure_items (id) ON DELETE CASCADE,
    outcome_id     UUID REFERENCES course.course_learning_outcomes (id) ON DELETE SET NULL,
    points_possible NUMERIC(10,2),
    counts_for_grade BOOLEAN NOT NULL DEFAULT FALSE,
    late_policy    TEXT NOT NULL DEFAULT 'accept'
                     CHECK (late_policy IN ('accept','accept_marked','reject')),
    enabled_by     UUID REFERENCES "user".users (id) ON DELETE SET NULL,
    enabled_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE course.content_tool_grade_links IS
    'CT.7: Opt-in gradebook bridge for scored content tools (off by default; per-instance consent).';

-- Optional org policy: when true, districts forbid non-assignment tool grading.
ALTER TABLE course.content_tool_settings
    ADD COLUMN IF NOT EXISTS grade_links_allowed BOOLEAN NOT NULL DEFAULT TRUE;

COMMENT ON COLUMN course.content_tool_settings.grade_links_allowed IS
    'CT.7: When false, instructors cannot enable content tool gradebook links in this course.';
