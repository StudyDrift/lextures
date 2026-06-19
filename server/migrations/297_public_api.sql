-- Plan 16.1: public REST API feature flag and request audit log.

CREATE SCHEMA IF NOT EXISTS api;

CREATE TABLE IF NOT EXISTS api.request_log (
    id          BIGSERIAL PRIMARY KEY,
    token_id    UUID REFERENCES auth.api_tokens(id) ON DELETE SET NULL,
    user_id     UUID REFERENCES "user".users(id) ON DELETE SET NULL,
    method      TEXT NOT NULL,
    path        TEXT NOT NULL,
    status      SMALLINT NOT NULL,
    latency_ms  INTEGER NOT NULL,
    ip_hash     TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS api_request_log_token_created_idx
    ON api.request_log (token_id, created_at DESC);
CREATE INDEX IF NOT EXISTS api_request_log_created_idx
    ON api.request_log (created_at DESC);

ALTER TABLE settings.platform_app_settings
    ADD COLUMN IF NOT EXISTS ff_public_api BOOLEAN;

COMMENT ON COLUMN settings.platform_app_settings.ff_public_api IS
    'Enables the versioned public REST API for third-party integrations (plan 16.1).';
