-- Per-student progress analytics (plan 9.1): materialized summary + instructor notes.

CREATE SCHEMA IF NOT EXISTS analytics;

CREATE TABLE analytics.student_progress_refresh (
    id SMALLINT PRIMARY KEY DEFAULT 1 CHECK (id = 1),
    refreshed_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

INSERT INTO analytics.student_progress_refresh (id, refreshed_at) VALUES (1, NOW())
ON CONFLICT (id) DO NOTHING;

CREATE MATERIALIZED VIEW analytics.student_progress AS
SELECT
    e.id AS enrollment_id,
    e.course_id,
    e.user_id,
    COALESCE(asub.cnt, 0)::int AS assignments_submitted,
    COALESCE(atot.cnt, 0)::int AS assignments_total,
    qavg.avg_quiz_score,
    COALESCE(mview.cnt, 0)::int AS module_views_count,
    COALESCE(mtot.cnt, 0)::int AS modules_total,
    la.last_active_at
FROM course.course_enrollments e
LEFT JOIN LATERAL (
    SELECT COUNT(*)::int AS cnt
    FROM course.course_structure_items csi
    INNER JOIN course.module_assignment_submissions mas
        ON mas.module_item_id = csi.id AND mas.submitted_by = e.user_id
    WHERE csi.course_id = e.course_id
      AND csi.kind = 'assignment'
      AND csi.published
      AND NOT csi.archived
) asub ON true
LEFT JOIN LATERAL (
    SELECT COUNT(*)::int AS cnt
    FROM course.course_structure_items csi
    WHERE csi.course_id = e.course_id
      AND csi.kind = 'assignment'
      AND csi.published
      AND NOT csi.archived
) atot ON true
LEFT JOIN LATERAL (
    SELECT AVG(qa.score_percent)::real AS avg_quiz_score
    FROM course.quiz_attempts qa
    WHERE qa.course_id = e.course_id
      AND qa.student_user_id = e.user_id
      AND qa.status = 'submitted'
      AND qa.score_percent IS NOT NULL
) qavg ON true
LEFT JOIN LATERAL (
    SELECT COUNT(DISTINCT ua.structure_item_id)::int AS cnt
    FROM "user".user_audit ua
    WHERE ua.user_id = e.user_id
      AND ua.course_id = e.course_id
      AND ua.event_kind = 'content_open'
      AND ua.structure_item_id IS NOT NULL
) mview ON true
LEFT JOIN LATERAL (
    SELECT COUNT(*)::int AS cnt
    FROM course.course_structure_items csi
    WHERE csi.course_id = e.course_id
      AND csi.kind = 'content_page'
      AND csi.published
      AND NOT csi.archived
) mtot ON true
LEFT JOIN LATERAL (
    SELECT MAX(ua.occurred_at) AS last_active_at
    FROM "user".user_audit ua
    WHERE ua.user_id = e.user_id AND ua.course_id = e.course_id
) la ON true
WHERE e.active = true;

CREATE UNIQUE INDEX idx_analytics_student_progress_enrollment
    ON analytics.student_progress (enrollment_id);

CREATE INDEX idx_analytics_student_progress_course
    ON analytics.student_progress (course_id);

CREATE TABLE analytics.instructor_progress_notes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    enrollment_id UUID NOT NULL REFERENCES course.course_enrollments (id) ON DELETE CASCADE,
    author_id UUID NOT NULL REFERENCES "user".users (id) ON DELETE CASCADE,
    note_text TEXT NOT NULL CHECK (char_length(note_text) >= 1 AND char_length(note_text) <= 10000),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_instructor_progress_notes_enrollment
    ON analytics.instructor_progress_notes (enrollment_id);

COMMENT ON MATERIALIZED VIEW analytics.student_progress IS
    'Cached per-enrollment progress metrics; refresh at most every 5 minutes (plan 9.1).';
COMMENT ON TABLE analytics.instructor_progress_notes IS
    'Private instructor notes on student progress; not visible to students (FERPA).';

REFRESH MATERIALIZED VIEW analytics.student_progress;
