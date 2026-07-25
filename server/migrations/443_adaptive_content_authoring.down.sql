-- AC.5 down: remove review metadata and catalog permissions.

DROP INDEX IF EXISTS course.idx_ac_variants_pending_review;

ALTER TABLE course.content_variants
    DROP COLUMN IF EXISTS human_edited,
    DROP COLUMN IF EXISTS reviewed_by,
    DROP COLUMN IF EXISTS reviewed_at,
    DROP COLUMN IF EXISTS review_note,
    DROP COLUMN IF EXISTS variant_version;

DELETE FROM "user".permissions
WHERE permission_string IN (
    'course:adaptive_content:review',
    'course:*:adaptive_content:review'
);
