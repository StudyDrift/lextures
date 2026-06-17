-- AI token/cost usage for Intelligence → Reports (plan 19.14 foundation).
CREATE TABLE IF NOT EXISTS analytics.ai_usage_log (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    user_id           UUID REFERENCES "user".users(id) ON DELETE SET NULL,
    course_id         UUID REFERENCES course.courses(id) ON DELETE SET NULL,
    feature           TEXT NOT NULL DEFAULT 'unknown',
    model             TEXT NOT NULL,
    prompt_tokens     INT NOT NULL DEFAULT 0 CHECK (prompt_tokens >= 0),
    completion_tokens INT NOT NULL DEFAULT 0 CHECK (completion_tokens >= 0),
    total_tokens      INT NOT NULL DEFAULT 0 CHECK (total_tokens >= 0),
    cost_usd          NUMERIC(14, 8) NOT NULL DEFAULT 0 CHECK (cost_usd >= 0),
    succeeded         BOOLEAN NOT NULL DEFAULT TRUE
);

CREATE INDEX IF NOT EXISTS idx_ai_usage_log_created_at
    ON analytics.ai_usage_log (created_at DESC);

CREATE INDEX IF NOT EXISTS idx_ai_usage_log_user_created
    ON analytics.ai_usage_log (user_id, created_at DESC)
    WHERE user_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_ai_usage_log_course_created
    ON analytics.ai_usage_log (course_id, created_at DESC)
    WHERE course_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_ai_usage_log_feature_created
    ON analytics.ai_usage_log (feature, created_at DESC);

COMMENT ON TABLE analytics.ai_usage_log IS
    'Per-call OpenRouter usage (tokens, estimated USD cost) for platform AI reports.';