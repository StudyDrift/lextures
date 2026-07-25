-- Companion to: 440_adaptive_content_profiles.sql
-- See docs/runbooks/database-migration-rollback.md

DROP TABLE IF EXISTS course.adaptive_content_unit_concepts;

ALTER TABLE course.adaptive_content_units
    DROP COLUMN IF EXISTS mastery_freshness_days,
    DROP COLUMN IF EXISTS trigger_mode;

ALTER TABLE course.adaptation_profiles
    DROP COLUMN IF EXISTS is_neutral,
    DROP COLUMN IF EXISTS axis_set,
    DROP COLUMN IF EXISTS modality_pref,
    DROP COLUMN IF EXISTS reading_level_pref,
    DROP COLUMN IF EXISTS target_bloom;
