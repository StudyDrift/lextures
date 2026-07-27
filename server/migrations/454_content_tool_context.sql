-- CT.6 — Grounded context service & web-link ingestion.

-- Org-scoped cache of ingested external sources (shared across courses in the org).
CREATE TABLE IF NOT EXISTS course.content_tool_link_sources (
    id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id             UUID REFERENCES tenant.organizations (id) ON DELETE CASCADE,
    url_hash           TEXT NOT NULL,          -- sha256 of the normalized URL
    url                TEXT NOT NULL,
    final_url          TEXT,                   -- after redirects
    content_type       TEXT,
    title              TEXT,
    lang               TEXT,
    extracted_text     TEXT,
    extraction_version INTEGER NOT NULL DEFAULT 1,
    byte_size          INTEGER,
    etag               TEXT,
    last_modified      TEXT,
    status             TEXT NOT NULL DEFAULT 'pending'
                         CHECK (status IN ('pending','ready','blocked','failed','unsupported')),
    error              TEXT,
    fetched_at         TIMESTAMPTZ,
    expires_at         TIMESTAMPTZ,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (org_id, url_hash, extraction_version)
);
CREATE INDEX IF NOT EXISTS idx_ctls_expires ON course.content_tool_link_sources (expires_at);
CREATE INDEX IF NOT EXISTS idx_ctls_org_status ON course.content_tool_link_sources (org_id, status);

COMMENT ON TABLE course.content_tool_link_sources IS
    'CT.6: Org-scoped cache of ingested external link sources for Content Tools context packs.';

-- Chunked passages for retrieval + citation.
CREATE TABLE IF NOT EXISTS course.content_tool_link_chunks (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    source_id   UUID NOT NULL REFERENCES course.content_tool_link_sources (id) ON DELETE CASCADE,
    ordinal     INTEGER NOT NULL,
    text        TEXT NOT NULL,
    token_count INTEGER NOT NULL DEFAULT 0,
    UNIQUE (source_id, ordinal)
);

COMMENT ON TABLE course.content_tool_link_chunks IS
    'CT.6: Chunked passages from ingested link sources for retrieval and citation.';

-- Which sources an activity uses, plus instructor overrides.
CREATE TABLE IF NOT EXISTS course.content_tool_activity_sources (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    course_id         UUID NOT NULL REFERENCES course.courses (id) ON DELETE CASCADE,
    structure_item_id UUID REFERENCES course.course_structure_items (id) ON DELETE CASCADE,
    source_id         UUID REFERENCES course.content_tool_link_sources (id) ON DELETE CASCADE,
    origin            TEXT NOT NULL CHECK (origin IN ('body_link','config_link','course_file')),
    course_file_id    UUID,
    excluded          BOOLEAN NOT NULL DEFAULT FALSE,
    excluded_by       UUID REFERENCES "user".users (id) ON DELETE SET NULL,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_ctas_item_source
    ON course.content_tool_activity_sources (structure_item_id, source_id)
    WHERE source_id IS NOT NULL AND course_file_id IS NULL;
CREATE UNIQUE INDEX IF NOT EXISTS idx_ctas_item_file
    ON course.content_tool_activity_sources (structure_item_id, course_file_id)
    WHERE course_file_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_ctas_item ON course.content_tool_activity_sources (structure_item_id, excluded);

COMMENT ON TABLE course.content_tool_activity_sources IS
    'CT.6: Activity-to-source links with instructor exclude overrides.';

-- Per-course AI budget and link-ingestion policy for content tools.
ALTER TABLE course.content_tool_settings
    ADD COLUMN IF NOT EXISTS monthly_ai_token_budget BIGINT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS daily_ai_calls_per_user INTEGER NOT NULL DEFAULT 50,
    ADD COLUMN IF NOT EXISTS link_ingestion_mode TEXT NOT NULL DEFAULT 'public'
        CHECK (link_ingestion_mode IN ('off','allowlist','public')),
    ADD COLUMN IF NOT EXISTS link_host_allowlist TEXT[] NOT NULL DEFAULT '{}';

COMMENT ON COLUMN course.content_tool_settings.monthly_ai_token_budget IS
    'CT.6: Per-course monthly AI token budget for content tools (0 = org default / unlimited).';
COMMENT ON COLUMN course.content_tool_settings.daily_ai_calls_per_user IS
    'CT.6: Per-user daily AI call cap for content tools.';
COMMENT ON COLUMN course.content_tool_settings.link_ingestion_mode IS
    'CT.6: Link ingestion policy: off | allowlist | public.';
COMMENT ON COLUMN course.content_tool_settings.link_host_allowlist IS
    'CT.6: Host allowlist when link_ingestion_mode = allowlist.';
