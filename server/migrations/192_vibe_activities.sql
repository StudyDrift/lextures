-- Vibe Activity module items: instructor-authored self-contained interactive HTML experiences
-- stored directly in the DB (transactional with the structure item) for reliability and simple
-- rendering via srcDoc in a sandboxed iframe. "Vibe coding" = prompt + AI (or manual) → live preview
-- → save as a first-class module item students can open and interact with.

CREATE TABLE course.module_vibe_activities (
    structure_item_id UUID PRIMARY KEY REFERENCES course.course_structure_items (id) ON DELETE CASCADE,
    html_content TEXT NOT NULL DEFAULT '',
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Extend the two kind constraints (pattern from 165_h5p_packages.sql and earlier)
ALTER TABLE course.course_structure_items DROP CONSTRAINT IF EXISTS course_structure_items_kind_check;
ALTER TABLE course.course_structure_items
  ADD CONSTRAINT course_structure_items_kind_check
  CHECK (
    kind IN (
      'module',
      'heading',
      'content_page',
      'assignment',
      'quiz',
      'external_link',
      'survey',
      'lti_link',
      'h5p',
      'vibe_activity'
    )
  );

ALTER TABLE course.course_structure_items DROP CONSTRAINT IF EXISTS course_structure_items_parent_child_kind_check;
ALTER TABLE course.course_structure_items
  ADD CONSTRAINT course_structure_items_parent_child_kind_check
  CHECK (
    parent_id IS NULL
    OR kind IN (
      'heading',
      'content_page',
      'assignment',
      'quiz',
      'external_link',
      'survey',
      'lti_link',
      'h5p',
      'vibe_activity'
    )
  );

-- Helpful index (PK already covers lookups by item, but explicit for clarity)
CREATE INDEX IF NOT EXISTS idx_module_vibe_activities_item ON course.module_vibe_activities (structure_item_id);
