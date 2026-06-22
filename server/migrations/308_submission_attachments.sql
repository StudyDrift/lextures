-- Multiple files per assignment submission (Canvas multi-attachment imports).
CREATE TABLE IF NOT EXISTS course.submission_attachments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    submission_id UUID NOT NULL REFERENCES course.module_assignment_submissions (id) ON DELETE CASCADE,
    file_id UUID NOT NULL REFERENCES course.course_files (id) ON DELETE CASCADE,
    sort_order INT NOT NULL DEFAULT 0 CHECK (sort_order >= 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT submission_attachments_unique_file UNIQUE (submission_id, file_id)
);

CREATE INDEX IF NOT EXISTS submission_attachments_submission_id_idx
    ON course.submission_attachments (submission_id, sort_order);

COMMENT ON TABLE course.submission_attachments IS
    'Ordered files attached to a module assignment submission; module_assignment_submissions.attachment_file_id remains the primary (first) file.';

-- Backfill existing single-file submissions.
INSERT INTO course.submission_attachments (submission_id, file_id, sort_order)
SELECT s.id, s.attachment_file_id, 0
FROM course.module_assignment_submissions s
WHERE s.attachment_file_id IS NOT NULL
ON CONFLICT (submission_id, file_id) DO NOTHING;