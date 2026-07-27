-- CT.4 down: drop reset tables and org retention column.

DROP TABLE IF EXISTS course.content_tool_reset_jobs;
DROP TABLE IF EXISTS course.content_tool_state_resets;

ALTER TABLE tenant.organizations
    DROP COLUMN IF EXISTS content_tool_state_retention_days;
