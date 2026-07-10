-- MC0: Official marketplace course authoring & provisioning foundation.
-- Ledger tables for idempotent provision + is_official attribution.

ALTER TABLE course.courses
    ADD COLUMN IF NOT EXISTS is_official BOOLEAN NOT NULL DEFAULT FALSE;

COMMENT ON COLUMN course.courses.is_official IS
    'When true, the course is a first-party platform-authored official course (MC0). Used for badge + delete-protection.';

CREATE INDEX IF NOT EXISTS idx_courses_is_official
    ON course.courses (is_official)
    WHERE is_official = TRUE;

-- Dedicated system publisher for official marketplace courses (not intended for login).
INSERT INTO "user".users (id, email, password_hash, display_name, account_type, org_id, login_blocked)
VALUES (
    'a0000000-0000-4000-8000-000000000003'::uuid,
    'publisher@system.lextures.invalid',
    '!',
    'Lextures Official',
    'system',
    (SELECT id FROM tenant.organizations WHERE slug = 'default' LIMIT 1),
    TRUE
)
ON CONFLICT (id) DO UPDATE SET
    email = EXCLUDED.email,
    display_name = EXCLUDED.display_name,
    account_type = EXCLUDED.account_type,
    login_blocked = TRUE;

-- Course-level provisioning ledger (slug → course_id).
CREATE TABLE IF NOT EXISTS settings.marketplace_courses (
    slug              TEXT PRIMARY KEY,
    course_id         UUID NOT NULL REFERENCES course.courses (id) ON DELETE CASCADE,
    content_version   INTEGER NOT NULL DEFAULT 0,
    provisioned_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_marketplace_courses_course_id
    ON settings.marketplace_courses (course_id);

COMMENT ON TABLE settings.marketplace_courses IS
    'Maps official marketplace course slugs to course rows for idempotent provisioning (MC0).';

-- Per-item slug → structure_item_id map (mirrors settings.intro_course_items).
CREATE TABLE IF NOT EXISTS settings.marketplace_course_items (
    course_slug       TEXT NOT NULL REFERENCES settings.marketplace_courses (slug) ON DELETE CASCADE,
    slug              TEXT NOT NULL,
    structure_item_id UUID NOT NULL REFERENCES course.course_structure_items (id) ON DELETE CASCADE,
    content_version   INTEGER NOT NULL DEFAULT 0,
    grade_policy      TEXT NULL,
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (course_slug, slug),
    CONSTRAINT marketplace_course_items_grade_policy_check CHECK (
        grade_policy IS NULL OR grade_policy IN ('quiz_autoscore', 'completion_full', 'grader_agent')
    )
);

CREATE INDEX IF NOT EXISTS idx_marketplace_course_items_structure_item_id
    ON settings.marketplace_course_items (structure_item_id);

CREATE INDEX IF NOT EXISTS idx_marketplace_course_items_course_slug
    ON settings.marketplace_course_items (course_slug);

COMMENT ON TABLE settings.marketplace_course_items IS
    'Maps official marketplace curriculum item slugs to course_structure_items for idempotent content sync (MC0).';
