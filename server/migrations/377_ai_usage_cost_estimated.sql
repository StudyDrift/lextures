-- AP.6: cost estimate flag, model alias, and provider report index.

ALTER TABLE analytics.ai_usage_log
  ADD COLUMN IF NOT EXISTS cost_estimated BOOLEAN NOT NULL DEFAULT false,
  ADD COLUMN IF NOT EXISTS model_alias TEXT;

COMMENT ON COLUMN analytics.ai_usage_log.cost_estimated IS
  'True when cost_usd was estimated from the local price table (provider omitted usage.cost).';

COMMENT ON COLUMN analytics.ai_usage_log.model_alias IS
  'Optional stable model alias at call time (e.g. text-fast); model column stores the resolved provider id.';

CREATE INDEX IF NOT EXISTS idx_ai_usage_log_provider_created
  ON analytics.ai_usage_log (provider, created_at DESC);

COMMENT ON TABLE analytics.ai_usage_log IS
  'Per-call AI usage (tokens, USD cost or estimate) for platform Intelligence reports (multi-provider).';
