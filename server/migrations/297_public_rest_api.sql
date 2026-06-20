-- Public REST/GraphQL API layer with request audit log (plan 16.1).

CREATE SCHEMA IF NOT EXISTS api;

CREATE TABLE IF NOT EXISTS api.request_log (
    id          BIGSERIAL PRIMARY KEY,
    token_id    UUID REFERENCES auth.api_tokens (id) ON DELETE SET NULL,
    user_id     UUID REFERENCES "user".users (id) ON DELETE SET NULL,
    method      TEXT NOT NULL,
    path        TEXT NOT NULL,
    status      SMALLINT NOT NULL,
    latency_ms  INTEGER NOT NULL,
    ip_hash     TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

COMMENT ON TABLE api.request_log IS
    'Authenticated public API request audit trail; IP stored as hash only (plan 16.1).';

CREATE INDEX IF NOT EXISTS idx_api_request_log_token_created
    ON api.request_log (token_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_api_request_log_created
    ON api.request_log (created_at DESC);

ALTER TABLE settings.platform_app_settings
    ADD COLUMN IF NOT EXISTS ff_public_api BOOLEAN,
    ADD COLUMN IF NOT EXISTS ff_api_docs BOOLEAN;

COMMENT ON COLUMN settings.platform_app_settings.ff_public_api IS
    'Enables the versioned public REST API for institutional integrations (plan 16.1).';

COMMENT ON COLUMN settings.platform_app_settings.ff_api_docs IS
    'Enables Swagger UI and ReDoc at /api/v1/docs and /api/v1/redoc (plan 16.1).';
