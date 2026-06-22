-- Plan 3.17 — Grade curving & scaling: feature flag, curve metadata, raw-score preservation.

ALTER TABLE settings.platform_app_settings
    ADD COLUMN IF NOT EXISTS ff_grade_curving BOOLEAN;

COMMENT ON COLUMN settings.platform_app_settings.ff_grade_curving IS
    'Enables instructor grade curving/scaling on assignments (plan 3.17).';

CREATE TABLE course.grade_curves (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    course_id        UUID NOT NULL REFERENCES course.courses (id) ON DELETE CASCADE,
    module_item_id   UUID REFERENCES course.course_structure_items (id) ON DELETE CASCADE,
    scope            TEXT NOT NULL CHECK (scope IN ('assessment', 'final')),
    section_id       UUID,
    method           TEXT NOT NULL CHECK (method IN
        ('flat_bonus', 'linear_scale', 'sqrt_curve', 'set_minimum', 'custom_mapping')),
    params_json      JSONB NOT NULL DEFAULT '{}',
    allow_above_max  BOOLEAN NOT NULL DEFAULT FALSE,
    applied_by       UUID NOT NULL REFERENCES "user".users (id) ON DELETE RESTRICT,
    applied_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    reverted_at      TIMESTAMPTZ
);

COMMENT ON TABLE course.grade_curves IS
    'Instructor-applied grade curve/scaling metadata (plan 3.17).';

CREATE INDEX idx_grade_curves_assessment ON course.grade_curves (module_item_id)
    WHERE reverted_at IS NULL AND scope = 'assessment';

CREATE INDEX idx_grade_curves_course ON course.grade_curves (course_id, applied_at DESC);

CREATE TABLE course.grade_curve_adjustments (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    curve_id      UUID NOT NULL REFERENCES course.grade_curves (id) ON DELETE CASCADE,
    student_id    UUID NOT NULL REFERENCES "user".users (id) ON DELETE CASCADE,
    raw_score     NUMERIC(8, 2) NOT NULL,
    adjusted_score NUMERIC(8, 2) NOT NULL,
    CONSTRAINT grade_curve_adjustments_unique_student UNIQUE (curve_id, student_id)
);

COMMENT ON TABLE course.grade_curve_adjustments IS
    'Per-student raw vs curved scores for reversible curve application (plan 3.17).';

CREATE INDEX idx_grade_curve_adjustments_curve ON course.grade_curve_adjustments (curve_id);

-- Extend grade audit actions (3.10) for 3.17.
ALTER TABLE course.grade_audit_events
    DROP CONSTRAINT IF EXISTS grade_audit_events_action_check;

ALTER TABLE course.grade_audit_events
    ADD CONSTRAINT grade_audit_events_action_check CHECK (action IN
        (
            'created', 'updated', 'excused', 'unexcused', 'posted', 'retracted', 'deleted',
            'revision_requested', 'resubmission_received', 'curved', 'curve_reverted'
        ));
