-- Personal API access keys with fine-grained scopes (plan 16.2).

CREATE SCHEMA IF NOT EXISTS auth;

CREATE TABLE IF NOT EXISTS auth.api_tokens (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_user_id   UUID NOT NULL REFERENCES "user".users (id) ON DELETE CASCADE,
    label           TEXT NOT NULL,
    token_hash      TEXT NOT NULL UNIQUE,
    token_prefix    CHAR(8) NOT NULL,
    scopes          TEXT[] NOT NULL DEFAULT '{}',
    expires_at      TIMESTAMPTZ,
    last_used_at    TIMESTAMPTZ,
    revoked_at      TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_api_tokens_prefix
    ON auth.api_tokens (token_prefix)
    WHERE revoked_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_api_tokens_owner
    ON auth.api_tokens (owner_user_id)
    WHERE revoked_at IS NULL;

COMMENT ON TABLE auth.api_tokens IS 'Personal API access keys for tools and MCP agents (plan 16.2).';
