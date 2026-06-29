-- Per-token rate-limit override for institutional API tokens (plan 17.6 FR-6).
-- NULL means "use the deployment default" (config.RateLimits.APITokenPerMin).

ALTER TABLE auth.api_tokens
    ADD COLUMN IF NOT EXISTS rate_limit_per_min INTEGER;

COMMENT ON COLUMN auth.api_tokens.rate_limit_per_min IS
    'Per-minute request quota override for this token; NULL uses the deployment default (plan 17.6 FR-6).';
