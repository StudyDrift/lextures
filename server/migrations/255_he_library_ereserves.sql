-- HE Library / E-Reserves integration (plan 14.10): Leganto, Alma catalog search, EZproxy.

-- Add library_resource as a new module item kind.
ALTER TABLE course.course_structure_items DROP CONSTRAINT IF EXISTS course_structure_items_kind_check;
ALTER TABLE course.course_structure_items
    ADD CONSTRAINT course_structure_items_kind_check
    CHECK (kind IN ('module', 'heading', 'content_page', 'assignment', 'quiz', 'external_link', 'h5p', 'survey', 'lti_link', 'vibe_activity', 'attendance', 'library_resource'));

ALTER TABLE course.course_structure_items DROP CONSTRAINT IF EXISTS course_structure_items_parent_child_kind_check;
ALTER TABLE course.course_structure_items
    ADD CONSTRAINT course_structure_items_parent_child_kind_check
    CHECK (parent_id IS NULL OR kind IN ('heading', 'content_page', 'assignment', 'quiz', 'external_link', 'h5p', 'survey', 'lti_link', 'vibe_activity', 'attendance', 'library_resource'));

-- Per-item metadata for library resources (Alma catalog items or Leganto reading lists).
CREATE TABLE course.module_library_resources (
    structure_item_id UUID PRIMARY KEY REFERENCES course.course_structure_items (id) ON DELETE CASCADE,
    resource_type     TEXT NOT NULL DEFAULT 'catalog_item'
                        CHECK (resource_type IN ('catalog_item', 'leganto_list')),
    -- External tool id for Leganto LTI deep links (nullable for catalog_item type).
    external_tool_id  UUID REFERENCES settings.lti_external_tools (id) ON DELETE SET NULL,
    -- Alma / Leganto identifiers.
    alma_mms_id       TEXT,
    leganto_list_id   TEXT,
    -- Bibliographic metadata cached from Alma (title, author, ISSN/ISBN, source).
    metadata          JSONB NOT NULL DEFAULT '{}',
    -- EZproxy-rewritten URL (populated when an EZproxy prefix is configured for the org).
    ezproxy_url       TEXT,
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- COUNTER-compatible click events for library resource items (privacy by design — no user_id).
CREATE TABLE course.library_link_events (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    item_id     UUID NOT NULL REFERENCES course.course_structure_items (id) ON DELETE CASCADE,
    course_id   UUID NOT NULL REFERENCES course.courses (id) ON DELETE CASCADE,
    accessed_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_library_link_events_item ON course.library_link_events (item_id);
CREATE INDEX idx_library_link_events_course ON course.library_link_events (course_id);

-- Per-platform HE library integration config (EZproxy prefix, Alma API key, domain patterns).
CREATE TABLE settings.he_library_config (
    id                  SERIAL PRIMARY KEY,
    ezproxy_prefix      TEXT NOT NULL DEFAULT '',
    domain_patterns     TEXT[] NOT NULL DEFAULT '{}',
    alma_api_base_url   TEXT NOT NULL DEFAULT '',
    alma_api_key_cipher TEXT NOT NULL DEFAULT '',
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Feature flag for the HE library / e-reserves integration (plan 14.10).
ALTER TABLE settings.platform_app_settings
    ADD COLUMN IF NOT EXISTS ff_library_integration BOOLEAN;
