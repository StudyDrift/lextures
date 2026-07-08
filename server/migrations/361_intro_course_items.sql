-- IC03: stable slug → structure_item_id map for idempotent intro course content sync.
CREATE TABLE IF NOT EXISTS settings.intro_course_items (
    slug              TEXT PRIMARY KEY,
    structure_item_id UUID NOT NULL REFERENCES course.course_structure_items (id) ON DELETE CASCADE,
    content_version   INTEGER NOT NULL DEFAULT 0,
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_intro_course_items_structure_item_id
    ON settings.intro_course_items (structure_item_id);

COMMENT ON TABLE settings.intro_course_items IS
    'Maps intro course curriculum item slugs to course_structure_items rows for idempotent content sync (IC03).';