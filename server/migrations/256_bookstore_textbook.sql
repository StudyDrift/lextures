-- Bookstore / Textbook linking (plan 14.11): Inclusive Access delivery via VitalSource
-- Bridge / RedShelf BookShelf LTI 1.3 deep links, opt-out banner, COUNTER launch events.

-- Add textbook_resource as a new module item kind.
ALTER TABLE course.course_structure_items DROP CONSTRAINT IF EXISTS course_structure_items_kind_check;
ALTER TABLE course.course_structure_items
    ADD CONSTRAINT course_structure_items_kind_check
    CHECK (kind IN ('module', 'heading', 'content_page', 'assignment', 'quiz', 'external_link', 'h5p', 'survey', 'lti_link', 'vibe_activity', 'attendance', 'library_resource', 'textbook_resource'));

ALTER TABLE course.course_structure_items DROP CONSTRAINT IF EXISTS course_structure_items_parent_child_kind_check;
ALTER TABLE course.course_structure_items
    ADD CONSTRAINT course_structure_items_parent_child_kind_check
    CHECK (parent_id IS NULL OR kind IN ('heading', 'content_page', 'assignment', 'quiz', 'external_link', 'h5p', 'survey', 'lti_link', 'vibe_activity', 'attendance', 'library_resource', 'textbook_resource'));

-- Per-item metadata for textbook resources (VitalSource / RedShelf LTI deep links).
CREATE TABLE course.module_textbook_resources (
    structure_item_id UUID PRIMARY KEY REFERENCES course.course_structure_items (id) ON DELETE CASCADE,
    -- Bookstore provider for this deep link.
    provider          TEXT NOT NULL DEFAULT 'vitalsource'
                        CHECK (provider IN ('vitalsource', 'redshelf')),
    -- External tool id for the bookstore LTI 1.3 deep link.
    external_tool_id  UUID REFERENCES settings.lti_external_tools (id) ON DELETE SET NULL,
    -- Bibliographic + deep-link metadata: {isbn, title, edition, publisher, chapter, pageRange}.
    metadata          JSONB NOT NULL DEFAULT '{}',
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Inclusive Access configuration per course (opt-out banner + required materials).
CREATE TABLE course.inclusive_access_courses (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    course_id    UUID NOT NULL UNIQUE REFERENCES course.courses (id) ON DELETE CASCADE,
    isbn         TEXT NOT NULL,
    title        TEXT NOT NULL,
    opt_out_url  TEXT NOT NULL,
    provider     TEXT NOT NULL DEFAULT 'vitalsource'
                  CHECK (provider IN ('vitalsource', 'redshelf')),
    enabled      BOOLEAN NOT NULL DEFAULT TRUE,
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- COUNTER-compatible textbook launch events (privacy by design — no user_id / PII).
CREATE TABLE course.textbook_launch_events (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    item_id     UUID NOT NULL REFERENCES course.course_structure_items (id) ON DELETE CASCADE,
    course_id   UUID NOT NULL REFERENCES course.courses (id) ON DELETE CASCADE,
    provider    TEXT NOT NULL DEFAULT 'vitalsource',
    accessed_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_textbook_launch_events_item ON course.textbook_launch_events (item_id);
CREATE INDEX idx_textbook_launch_events_course ON course.textbook_launch_events (course_id);

-- Platform-wide bookstore integration config (default provider + registered LTI tools).
CREATE TABLE settings.bookstore_config (
    id                  SERIAL PRIMARY KEY,
    default_provider    TEXT NOT NULL DEFAULT 'vitalsource'
                          CHECK (default_provider IN ('vitalsource', 'redshelf')),
    vitalsource_tool_id UUID REFERENCES settings.lti_external_tools (id) ON DELETE SET NULL,
    redshelf_tool_id    UUID REFERENCES settings.lti_external_tools (id) ON DELETE SET NULL,
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Feature flag for the bookstore / textbook integration (plan 14.11).
ALTER TABLE settings.platform_app_settings
    ADD COLUMN IF NOT EXISTS ff_bookstore_integration BOOLEAN;
