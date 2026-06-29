-- Rollback for 342_api_token_rate_limit.sql
ALTER TABLE auth.api_tokens DROP COLUMN IF EXISTS rate_limit_per_min;
