-- AC.5 — Instructor authoring & human-in-the-loop approval: review metadata + review-only capability.

ALTER TABLE course.content_variants
    ADD COLUMN IF NOT EXISTS human_edited BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS reviewed_by UUID REFERENCES "user".users (id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS reviewed_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS review_note TEXT,
    ADD COLUMN IF NOT EXISTS variant_version INTEGER NOT NULL DEFAULT 1;

COMMENT ON COLUMN course.content_variants.human_edited IS
    'AC.5: True when an instructor edited the variant body before/while approving.';
COMMENT ON COLUMN course.content_variants.reviewed_by IS
    'AC.5: User who last approved or rejected this variant.';
COMMENT ON COLUMN course.content_variants.reviewed_at IS
    'AC.5: Timestamp of last approve/reject decision.';
COMMENT ON COLUMN course.content_variants.review_note IS
    'AC.5: Optional reason for reject (or note on approve).';
COMMENT ON COLUMN course.content_variants.variant_version IS
    'AC.5: Optimistic concurrency token; clients send expectedVariantVersion on review mutations.';

CREATE INDEX IF NOT EXISTS idx_ac_variants_pending_review
    ON course.content_variants (unit_id, status)
    WHERE status = 'pending_review';

-- Catalog permission (template). Concrete course grants use course:{code}:adaptive_content:review.
INSERT INTO "user".permissions (permission_string, description)
VALUES (
    'course:adaptive_content:review',
    'Approve or reject adaptive content variants without changing course-level adaptive settings.'
)
ON CONFLICT (permission_string) DO NOTHING;

INSERT INTO "user".permissions (permission_string, description)
VALUES (
    'course:*:adaptive_content:review',
    'Approve or reject adaptive content variants in any course (wildcard).'
)
ON CONFLICT (permission_string) DO NOTHING;
