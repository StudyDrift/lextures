ALTER TABLE course.content_tool_settings
    DROP COLUMN IF EXISTS grade_links_allowed;

DROP TABLE IF EXISTS course.content_tool_grade_links;
DROP TABLE IF EXISTS analytics.content_tool_daily_rollups;
DROP TABLE IF EXISTS analytics.content_tool_state_summaries;
