-- Plan 19.16 — Instructor-authored grading agent (SpeedGrader).

CREATE SCHEMA IF NOT EXISTS assessment;

CREATE TYPE assessment.grading_agent_status AS ENUM ('draft', 'accepted', 'archived');
CREATE TYPE assessment.grading_agent_run_scope AS ENUM ('current', 'ungraded', 'all', 'auto');
CREATE TYPE assessment.grading_agent_item_status AS ENUM ('suggested', 'applied', 'skipped', 'failed', 'overridden');

CREATE TABLE assessment.grading_agent_configs (
    id                         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    course_id                  UUID NOT NULL REFERENCES course.courses (id) ON DELETE CASCADE,
    module_item_id             UUID NOT NULL REFERENCES course.course_structure_items (id) ON DELETE CASCADE,
    status                     assessment.grading_agent_status NOT NULL DEFAULT 'draft',
    prompt                     TEXT NOT NULL,
    include_assignment_content BOOLEAN NOT NULL DEFAULT FALSE,
    include_rubric             BOOLEAN NOT NULL DEFAULT FALSE,
    model_id                   TEXT,
    auto_grade_new             BOOLEAN NOT NULL DEFAULT FALSE,
    post_policy                TEXT NOT NULL DEFAULT 'unposted',
    confidence_floor           NUMERIC CHECK (confidence_floor BETWEEN 0 AND 1),
    created_by                 UUID NOT NULL REFERENCES "user".users (id),
    created_at                 TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at                 TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (module_item_id)
);

COMMENT ON TABLE assessment.grading_agent_configs IS
    'Per-assignment instructor-authored grading agent configuration (plan 19.16).';

CREATE TABLE assessment.grading_agent_runs (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    config_id       UUID NOT NULL REFERENCES assessment.grading_agent_configs (id) ON DELETE CASCADE,
    scope           assessment.grading_agent_run_scope NOT NULL,
    initiated_by    UUID REFERENCES "user".users (id),
    total_count     INT NOT NULL DEFAULT 0,
    completed_count INT NOT NULL DEFAULT 0,
    failed_count    INT NOT NULL DEFAULT 0,
    status          TEXT NOT NULL DEFAULT 'queued',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    finished_at     TIMESTAMPTZ
);

CREATE INDEX idx_grading_agent_runs_config ON assessment.grading_agent_runs (config_id);

CREATE TABLE assessment.grading_agent_results (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    run_id           UUID REFERENCES assessment.grading_agent_runs (id) ON DELETE CASCADE,
    config_id        UUID NOT NULL REFERENCES assessment.grading_agent_configs (id) ON DELETE CASCADE,
    submission_id    UUID NOT NULL REFERENCES course.module_assignment_submissions (id) ON DELETE CASCADE,
    is_dry_run       BOOLEAN NOT NULL DEFAULT FALSE,
    suggested_points NUMERIC,
    suggested_rubric JSONB,
    comment          TEXT,
    confidence       NUMERIC CHECK (confidence BETWEEN 0 AND 1),
    status           assessment.grading_agent_item_status NOT NULL DEFAULT 'suggested',
    applied_grade_id UUID,
    model_id         TEXT,
    prompt_tokens    INT,
    completion_tokens INT,
    cost_usd         NUMERIC(14, 8),
    error            TEXT,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (run_id, submission_id)
);

CREATE INDEX idx_grading_agent_results_submission ON assessment.grading_agent_results (submission_id);
CREATE INDEX idx_grading_agent_results_config ON assessment.grading_agent_results (config_id);

ALTER TABLE course.course_grades
    ADD COLUMN IF NOT EXISTS graded_by_ai BOOLEAN NOT NULL DEFAULT FALSE;

COMMENT ON COLUMN course.course_grades.graded_by_ai IS
    'True when the grade was drafted by an AI grading agent and reviewed by staff (plan 19.16 / 19.3).';

ALTER TABLE settings.platform_app_settings
    ADD COLUMN IF NOT EXISTS grader_agent_enabled BOOLEAN;

COMMENT ON COLUMN settings.platform_app_settings.grader_agent_enabled IS
    'Enables the instructor-authored grading agent in SpeedGrader (plan 19.16).';