-- CC.6: Accessibility / launch markers and outbound link-health cache.

ALTER TABLE course.courses
    ADD COLUMN IF NOT EXISTS a11y_reviewed_at   TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS student_preview_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS last_export_at     TIMESTAMPTZ;

COMMENT ON COLUMN course.courses.a11y_reviewed_at IS
    'Set when course staff review/save accessibility settings; drives a11y.enforcement-settings.';
COMMENT ON COLUMN course.courses.student_preview_at IS
    'Set when staff use View as: Student; drives launch.student-preview.';
COMMENT ON COLUMN course.courses.last_export_at IS
    'Set when a course export is generated; drives launch.backup-export.';

CREATE TABLE IF NOT EXISTS course.course_checklist_link_health (
    course_id    UUID NOT NULL REFERENCES course.courses (id) ON DELETE CASCADE,
    url_hash     BYTEA NOT NULL,
    url          TEXT NOT NULL,
    status_code  INT,
    result       TEXT NOT NULL CHECK (result IN ('ok','dead','error','skipped')),
    reason       TEXT NOT NULL DEFAULT '',
    checked_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (course_id, url_hash)
);

CREATE INDEX IF NOT EXISTS idx_checklist_link_health_checked
    ON course.course_checklist_link_health (checked_at);

COMMENT ON TABLE course.course_checklist_link_health IS
    'Cached outbound link-health results for checklist item links.external-health (30-day retention).';
