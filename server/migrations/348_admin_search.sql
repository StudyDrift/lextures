-- Plan 18.4: Org-wide admin search — pg_trgm, user FTS indexes, feature flag, search log.

CREATE EXTENSION IF NOT EXISTS pg_trgm;

-- User full-text search (name + email).
ALTER TABLE "user".users
    ADD COLUMN IF NOT EXISTS search_vector tsvector
    GENERATED ALWAYS AS (
        to_tsvector(
            'english',
            coalesce(first_name, '') || ' ' ||
            coalesce(last_name, '') || ' ' ||
            coalesce(display_name, '') || ' ' ||
            coalesce(email, '')
        )
    ) STORED;

CREATE INDEX IF NOT EXISTS idx_users_search_vector_gin
    ON "user".users USING gin (search_vector);

CREATE INDEX IF NOT EXISTS idx_users_email_trgm
    ON "user".users USING gin (email gin_trgm_ops);

-- Extend course search vector to include description (drop/recreate generated column).
DROP INDEX IF EXISTS idx_courses_search_vector_gin;
ALTER TABLE course.courses DROP COLUMN IF EXISTS search_vector;

ALTER TABLE course.courses
    ADD COLUMN search_vector tsvector
    GENERATED ALWAYS AS (
        to_tsvector(
            'english',
            coalesce(course_code, '') || ' ' ||
            coalesce(title, '') || ' ' ||
            coalesce(description, '')
        )
    ) STORED;

CREATE INDEX IF NOT EXISTS idx_courses_search_vector_gin
    ON course.courses USING gin (search_vector);

CREATE INDEX IF NOT EXISTS idx_courses_title_trgm
    ON course.courses USING gin (title gin_trgm_ops);

-- Admin search query log (PII-scrubbed query text stored by the application).
CREATE TABLE IF NOT EXISTS compliance.admin_search_log (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    actor_id        UUID NOT NULL REFERENCES "user".users (id) ON DELETE CASCADE,
    org_id          UUID NOT NULL REFERENCES tenant.organizations (id) ON DELETE CASCADE,
    query_scrubbed  TEXT NOT NULL,
    user_count      INTEGER NOT NULL DEFAULT 0,
    course_count    INTEGER NOT NULL DEFAULT 0,
    content_count   INTEGER NOT NULL DEFAULT 0,
    took_ms         INTEGER NOT NULL DEFAULT 0,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_admin_search_log_org_created
    ON compliance.admin_search_log (org_id, created_at DESC);

COMMENT ON TABLE compliance.admin_search_log IS
    'Org-wide admin search queries with PII-scrubbed text (plan 18.4 FR-10).';

ALTER TABLE settings.platform_app_settings
    ADD COLUMN IF NOT EXISTS admin_search_enabled BOOLEAN;

COMMENT ON COLUMN settings.platform_app_settings.admin_search_enabled IS
    'Plan 18.4: Enables org-wide admin search (/api/v1/admin/search) for org_admin and global admin users.';
