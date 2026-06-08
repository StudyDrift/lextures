-- Plan 14.5 — Final Grade Roll-Up to Registrar (CSV Export).

CREATE TABLE course.final_grade_submissions (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    course_id         UUID NOT NULL REFERENCES course.courses (id) ON DELETE CASCADE,
    enrollment_id     UUID NOT NULL REFERENCES course.course_enrollments (id) ON DELETE CASCADE,
    submitted_by      UUID NOT NULL REFERENCES "user".users (id) ON DELETE CASCADE,
    computed_grade    TEXT NOT NULL,
    final_grade       TEXT NOT NULL,
    override_reason   TEXT,
    submission_method TEXT NOT NULL DEFAULT 'csv'
        CHECK (submission_method IN ('csv', 'ags')),
    submitted_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    sis_ack_at        TIMESTAMPTZ
);

COMMENT ON TABLE course.final_grade_submissions IS
    'Audit log of final grade submissions per enrollment (plan 14.5). One row per submission event; re-submissions add new rows.';

COMMENT ON COLUMN course.final_grade_submissions.computed_grade IS
    'Grade computed from gradebook at submission time (before any instructor override).';

COMMENT ON COLUMN course.final_grade_submissions.final_grade IS
    'Actual grade submitted — equals computed_grade unless instructor overrode.';

COMMENT ON COLUMN course.final_grade_submissions.override_reason IS
    'Required non-NULL reason when final_grade differs from computed_grade.';

COMMENT ON COLUMN course.final_grade_submissions.submission_method IS
    'csv = downloaded CSV; ags = LTI AGS passback.';

COMMENT ON COLUMN course.final_grade_submissions.sis_ack_at IS
    'Timestamp the SIS acknowledged the AGS passback (NULL until confirmed).';

CREATE INDEX idx_final_grade_submissions_course
    ON course.final_grade_submissions (course_id, submitted_at DESC);

CREATE INDEX idx_final_grade_submissions_enrollment
    ON course.final_grade_submissions (enrollment_id, submitted_at DESC);

-- Feature flag for final grade submission workflow (plan 14.5).
ALTER TABLE settings.platform_app_settings
    ADD COLUMN IF NOT EXISTS ff_grade_submission BOOLEAN;

COMMENT ON COLUMN settings.platform_app_settings.ff_grade_submission IS
    'Enables the final grade roll-up, review, and SIS export workflow (plan 14.5).';
