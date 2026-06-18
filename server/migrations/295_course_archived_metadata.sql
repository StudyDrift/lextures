ALTER TABLE course.courses
    ADD COLUMN IF NOT EXISTS archived_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS archived_by_user_id UUID REFERENCES "user".users (id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_courses_archived_org
    ON course.courses (org_id, archived_at DESC)
    WHERE archived = TRUE;

COMMENT ON COLUMN course.courses.archived_at IS
    'When the course was archived (soft-deleted from catalogs).';
COMMENT ON COLUMN course.courses.archived_by_user_id IS
    'User who archived the course.';