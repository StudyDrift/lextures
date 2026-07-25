-- Companion to: 441_adaptive_content_generation.sql

DELETE FROM settings.system_prompts WHERE key = 'adaptive_content_variant';

DROP TABLE IF EXISTS course.adaptive_content_key_terms;

ALTER TABLE course.adaptive_content_units
    DROP CONSTRAINT IF EXISTS adaptive_content_units_min_fidelity_check;

ALTER TABLE course.adaptive_content_units
    DROP COLUMN IF EXISTS content_version,
    DROP COLUMN IF EXISTS min_fidelity;

ALTER TABLE course.content_variants
    DROP COLUMN IF EXISTS prompt_version,
    DROP COLUMN IF EXISTS content_version,
    DROP COLUMN IF EXISTS prompt_tokens,
    DROP COLUMN IF EXISTS completion_tokens,
    DROP COLUMN IF EXISTS a11y_flags;
