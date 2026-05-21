-- Interactive H5P content (plan 8.12)

CREATE SCHEMA IF NOT EXISTS content;

CREATE TABLE content.h5p_packages (
  id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  storage_object_id UUID NOT NULL REFERENCES storage.objects(id) ON DELETE CASCADE,
  structure_item_id UUID REFERENCES course.course_structure_items(id) ON DELETE CASCADE,
  course_id         UUID NOT NULL REFERENCES course.courses(id) ON DELETE CASCADE,
  title             TEXT NOT NULL,
  content_type      TEXT NOT NULL,
  h5p_version       TEXT,
  manifest_json     JSONB NOT NULL DEFAULT '{}'::jsonb,
  assets_prefix     TEXT NOT NULL,
  extract_status    TEXT NOT NULL DEFAULT 'pending'
    CHECK (extract_status IN ('pending', 'ready', 'failed')),
  extract_error     TEXT,
  created_at        TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX ON content.h5p_packages (course_id);
CREATE INDEX ON content.h5p_packages (structure_item_id);
CREATE INDEX ON content.h5p_packages (extract_status) WHERE extract_status = 'pending';

CREATE TABLE content.h5p_completions (
  id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  package_id      UUID NOT NULL REFERENCES content.h5p_packages(id) ON DELETE CASCADE,
  user_id         UUID NOT NULL REFERENCES "user".users(id) ON DELETE CASCADE,
  status          TEXT NOT NULL DEFAULT 'not_started'
    CHECK (status IN ('not_started', 'in_progress', 'completed', 'passed', 'failed')),
  score_raw       REAL,
  score_max       REAL,
  xapi_statement  JSONB,
  updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (package_id, user_id)
);

CREATE INDEX ON content.h5p_completions (package_id, status);

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
      'h5p'
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
      'h5p'
    )
  );
