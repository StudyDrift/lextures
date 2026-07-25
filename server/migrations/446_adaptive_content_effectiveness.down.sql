-- AC.7 down: remove effectiveness cache and outcome extensions.

ALTER TABLE analytics.outcomes_report_student
    DROP COLUMN IF EXISTS adaptive_arm;

DROP TABLE IF EXISTS analytics.adaptive_content_effectiveness_by_variant;
DROP TABLE IF EXISTS analytics.adaptive_content_effectiveness_by_mode;
DROP TABLE IF EXISTS analytics.adaptive_content_effectiveness;

DROP INDEX IF EXISTS course.idx_ac_outcomes_measured;

ALTER TABLE course.adaptation_outcomes
    DROP COLUMN IF EXISTS post_attempt_id,
    DROP COLUMN IF EXISTS was_holdout,
    DROP COLUMN IF EXISTS emphasis_mode;
