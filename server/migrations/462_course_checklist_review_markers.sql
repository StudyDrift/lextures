-- CC.3: Review markers for course features and syllabus acceptance decisions.

ALTER TABLE course.courses
    ADD COLUMN IF NOT EXISTS features_reviewed_at TIMESTAMPTZ;

ALTER TABLE course.course_syllabus
    ADD COLUMN IF NOT EXISTS acceptance_decided_at TIMESTAMPTZ;

COMMENT ON COLUMN course.courses.features_reviewed_at IS
    'Set when course feature switches are saved; drives checklist item course.features-reviewed.';
COMMENT ON COLUMN course.course_syllabus.acceptance_decided_at IS
    'Set when require_syllabus_acceptance is explicitly chosen; drives syllabus.acceptance-decision.';
