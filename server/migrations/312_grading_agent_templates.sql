-- Course-scoped reusable grading agent workflow templates.

CREATE TABLE assessment.grading_agent_templates (
    id                         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    course_id                  UUID NOT NULL REFERENCES course.courses (id) ON DELETE CASCADE,
    name                       TEXT NOT NULL,
    prompt                     TEXT NOT NULL,
    include_assignment_content BOOLEAN NOT NULL DEFAULT FALSE,
    include_rubric             BOOLEAN NOT NULL DEFAULT FALSE,
    workflow_graph             JSONB NOT NULL,
    created_by                 UUID NOT NULL REFERENCES "user".users (id),
    created_at                 TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at                 TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE assessment.grading_agent_templates IS
    'Reusable grading agent workflow graphs saved as templates for a course.';

CREATE INDEX idx_grading_agent_templates_course
    ON assessment.grading_agent_templates (course_id, updated_at DESC);
