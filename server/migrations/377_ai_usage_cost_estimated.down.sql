-- Companion to: 377_ai_usage_cost_estimated.sql

DROP INDEX IF EXISTS analytics.idx_ai_usage_log_provider_created;

ALTER TABLE analytics.ai_usage_log
  DROP COLUMN IF EXISTS model_alias,
  DROP COLUMN IF EXISTS cost_estimated;
