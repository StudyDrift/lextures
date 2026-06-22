-- SCORM / xAPI / cmi5 content package ingestion (plan 2.14)

CREATE TABLE content.scorm_packages (
  id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  storage_object_id UUID NOT NULL REFERENCES storage.objects(id) ON DELETE CASCADE,
  structure_item_id UUID REFERENCES course.course_structure_items(id) ON DELETE CASCADE,
  course_id         UUID NOT NULL REFERENCES course.courses(id) ON DELETE CASCADE,
  title             TEXT NOT NULL,
  package_type      TEXT NOT NULL
    CHECK (package_type IN ('scorm12', 'scorm2004', 'cmi5')),
  manifest_json     JSONB NOT NULL DEFAULT '{}'::jsonb,
  assets_prefix     TEXT NOT NULL,
  extract_status    TEXT NOT NULL DEFAULT 'pending'
    CHECK (extract_status IN ('pending', 'ready', 'failed')),
  extract_error     TEXT,
  created_at        TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX ON content.scorm_packages (course_id);
CREATE INDEX ON content.scorm_packages (structure_item_id);
CREATE INDEX ON content.scorm_packages (extract_status) WHERE extract_status = 'pending';

CREATE TABLE content.scorm_scos (
  id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  package_id      UUID NOT NULL REFERENCES content.scorm_packages(id) ON DELETE CASCADE,
  identifier      TEXT NOT NULL,
  title           TEXT NOT NULL,
  launch_href     TEXT NOT NULL,
  sequencing_json JSONB NOT NULL DEFAULT '{}'::jsonb,
  mastery_score   REAL,
  UNIQUE (package_id, identifier)
);

CREATE INDEX ON content.scorm_scos (package_id);

CREATE TABLE content.scorm_registrations (
  id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  sco_id             UUID NOT NULL REFERENCES content.scorm_scos(id) ON DELETE CASCADE,
  enrollment_id      UUID NOT NULL REFERENCES course.course_enrollments(id) ON DELETE CASCADE,
  attempt_no         INT NOT NULL DEFAULT 1,
  completion_status  TEXT NOT NULL DEFAULT 'not attempted',
  success_status     TEXT NOT NULL DEFAULT 'unknown',
  score_scaled       REAL,
  score_raw          REAL,
  score_max          REAL,
  total_time_seconds INT NOT NULL DEFAULT 0,
  suspend_data       TEXT NOT NULL DEFAULT '',
  location           TEXT NOT NULL DEFAULT '',
  updated_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (sco_id, enrollment_id, attempt_no)
);

CREATE INDEX ON content.scorm_registrations (sco_id, enrollment_id);

CREATE TABLE content.scorm_rte_events (
  id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  registration_id UUID NOT NULL REFERENCES content.scorm_registrations(id) ON DELETE CASCADE,
  verb            TEXT NOT NULL,
  payload_json    JSONB NOT NULL DEFAULT '{}'::jsonb,
  occurred_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX ON content.scorm_rte_events (registration_id, occurred_at);

ALTER TABLE course.course_structure_items DROP CONSTRAINT IF EXISTS course_structure_items_kind_check;
ALTER TABLE course.course_structure_items
  ADD CONSTRAINT course_structure_items_kind_check
  CHECK (kind IN (
    'module', 'heading', 'content_page', 'assignment', 'quiz', 'external_link',
    'h5p', 'survey', 'lti_link', 'vibe_activity', 'attendance', 'library_resource',
    'textbook_resource', 'scorm'
  ));

ALTER TABLE course.course_structure_items DROP CONSTRAINT IF EXISTS course_structure_items_parent_child_kind_check;
ALTER TABLE course.course_structure_items
  ADD CONSTRAINT course_structure_items_parent_child_kind_check
  CHECK (parent_id IS NULL OR kind IN (
    'heading', 'content_page', 'assignment', 'quiz', 'external_link',
    'h5p', 'survey', 'lti_link', 'vibe_activity', 'attendance', 'library_resource',
    'textbook_resource', 'scorm'
  ));

ALTER TABLE settings.platform_app_settings
  ADD COLUMN IF NOT EXISTS scorm_ingestion_enabled BOOLEAN;
