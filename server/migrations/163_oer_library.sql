-- OER library (plan 8.9): search cache, provider settings, external link attribution.

CREATE SCHEMA IF NOT EXISTS content;

CREATE TABLE content.oer_search_cache (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    provider     TEXT NOT NULL,
    query_hash   TEXT NOT NULL,
    results_json JSONB NOT NULL,
    fetched_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at   TIMESTAMPTZ NOT NULL
);

CREATE UNIQUE INDEX oer_search_cache_provider_query_hash
    ON content.oer_search_cache (provider, query_hash);
CREATE INDEX oer_search_cache_expires_at ON content.oer_search_cache (expires_at);

ALTER TABLE course.module_external_links
    ADD COLUMN IF NOT EXISTS license_spdx TEXT,
    ADD COLUMN IF NOT EXISTS attribution_text TEXT,
    ADD COLUMN IF NOT EXISTS oer_provider TEXT;

CREATE TABLE IF NOT EXISTS settings.oer_provider_settings (
    provider   TEXT PRIMARY KEY,
    enabled    BOOLEAN NOT NULL DEFAULT TRUE,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

INSERT INTO settings.oer_provider_settings (provider, enabled) VALUES
    ('oer_commons', TRUE),
    ('merlot', TRUE),
    ('openstax', TRUE)
ON CONFLICT (provider) DO NOTHING;
