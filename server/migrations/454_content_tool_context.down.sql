ALTER TABLE course.content_tool_settings
    DROP COLUMN IF EXISTS link_host_allowlist,
    DROP COLUMN IF EXISTS link_ingestion_mode,
    DROP COLUMN IF EXISTS daily_ai_calls_per_user,
    DROP COLUMN IF EXISTS monthly_ai_token_budget;

DROP TABLE IF EXISTS course.content_tool_activity_sources;
DROP TABLE IF EXISTS course.content_tool_link_chunks;
DROP TABLE IF EXISTS course.content_tool_link_sources;
