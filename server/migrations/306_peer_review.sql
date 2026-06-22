-- Plan 3.15 — Peer review & peer assessment.

CREATE TYPE course.peer_review_anonymity AS ENUM ('double_blind', 'reviewer_anon', 'named');
CREATE TYPE course.peer_review_grade_mode AS ENUM ('none', 'score_only', 'weighted_blend');
CREATE TYPE course.peer_review_aggregation AS ENUM ('mean', 'median', 'trimmed');
CREATE TYPE course.peer_review_allocation_status AS ENUM ('assigned', 'in_progress', 'submitted', 'expired');

CREATE TABLE IF NOT EXISTS course.peer_review_configs (
    id                   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    assignment_id        UUID NOT NULL UNIQUE REFERENCES course.course_structure_items (id) ON DELETE CASCADE,
    reviews_per_reviewer INT NOT NULL DEFAULT 3 CHECK (reviews_per_reviewer >= 1 AND reviews_per_reviewer <= 20),
    anonymity            course.peer_review_anonymity NOT NULL DEFAULT 'double_blind',
    opens_at             TIMESTAMPTZ,
    closes_at            TIMESTAMPTZ,
    grade_mode           course.peer_review_grade_mode NOT NULL DEFAULT 'none',
    blend_weight         NUMERIC(5, 4) NOT NULL DEFAULT 0.3
        CHECK (blend_weight >= 0 AND blend_weight <= 1),
    aggregation          course.peer_review_aggregation NOT NULL DEFAULT 'median',
    exclude_same_group   BOOLEAN NOT NULL DEFAULT TRUE,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE course.peer_review_configs IS
    'Instructor peer-review settings per assignment (plan 3.15).';

CREATE TABLE IF NOT EXISTS course.peer_review_allocations (
    id                     UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    config_id              UUID NOT NULL REFERENCES course.peer_review_configs (id) ON DELETE CASCADE,
    reviewer_enrollment_id UUID NOT NULL REFERENCES course.course_enrollments (id) ON DELETE CASCADE,
    target_submission_id   UUID NOT NULL REFERENCES course.module_assignment_submissions (id) ON DELETE CASCADE,
    status                 course.peer_review_allocation_status NOT NULL DEFAULT 'assigned',
    assigned_at            TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (config_id, reviewer_enrollment_id, target_submission_id)
);

CREATE INDEX IF NOT EXISTS idx_peer_review_allocations_config
    ON course.peer_review_allocations (config_id);
CREATE INDEX IF NOT EXISTS idx_peer_review_allocations_reviewer
    ON course.peer_review_allocations (reviewer_enrollment_id);
CREATE INDEX IF NOT EXISTS idx_peer_review_allocations_target
    ON course.peer_review_allocations (target_submission_id);

COMMENT ON TABLE course.peer_review_allocations IS
    'Reviewer-to-submission assignments for peer review (plan 3.15).';

CREATE TABLE IF NOT EXISTS course.peer_reviews (
    id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    allocation_id      UUID NOT NULL UNIQUE REFERENCES course.peer_review_allocations (id) ON DELETE CASCADE,
    score              DOUBLE PRECISION CHECK (score IS NULL OR (score >= 0 AND score <= 1e9)),
    rubric_scores_json JSONB,
    comments           TEXT,
    submitted_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE course.peer_reviews IS
    'Submitted peer review scores and feedback (plan 3.15).';

CREATE TABLE IF NOT EXISTS course.peer_review_helpfulness (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    peer_review_id      UUID NOT NULL REFERENCES course.peer_reviews (id) ON DELETE CASCADE,
    rater_enrollment_id UUID NOT NULL REFERENCES course.course_enrollments (id) ON DELETE CASCADE,
    rating              SMALLINT NOT NULL CHECK (rating >= 1 AND rating <= 5),
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (peer_review_id, rater_enrollment_id)
);

COMMENT ON TABLE course.peer_review_helpfulness IS
    'Reviewee back-evaluation of received peer feedback helpfulness (plan 3.15).';

CREATE TABLE IF NOT EXISTS course.team_peer_evaluations (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    group_id            UUID NOT NULL REFERENCES course.enrollment_groups (id) ON DELETE CASCADE,
    rater_enrollment_id UUID NOT NULL REFERENCES course.course_enrollments (id) ON DELETE CASCADE,
    ratee_enrollment_id UUID NOT NULL REFERENCES course.course_enrollments (id) ON DELETE CASCADE,
    contribution_score  SMALLINT NOT NULL CHECK (contribution_score >= 1 AND contribution_score <= 5),
    comment             TEXT,
    submitted_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (group_id, rater_enrollment_id, ratee_enrollment_id),
    CHECK (rater_enrollment_id <> ratee_enrollment_id)
);

CREATE INDEX IF NOT EXISTS idx_team_peer_evaluations_group
    ON course.team_peer_evaluations (group_id);

COMMENT ON TABLE course.team_peer_evaluations IS
    'CATME-style team member contribution ratings visible to instructors only (plan 3.15).';

ALTER TABLE settings.platform_app_settings
    ADD COLUMN IF NOT EXISTS ff_peer_review BOOLEAN;

COMMENT ON COLUMN settings.platform_app_settings.ff_peer_review IS
    'When true, enables peer review configuration, allocation, and student review workspace (plan 3.15).';
