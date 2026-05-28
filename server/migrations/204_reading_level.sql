-- Plan 11.6: Reading-level adaptation (Flesch-Kincaid scoring + AI simplification cache).

CREATE SCHEMA IF NOT EXISTS i18n;

ALTER TABLE course.module_content_pages
    ADD COLUMN IF NOT EXISTS reading_level_fkgl NUMERIC(4, 1),
    ADD COLUMN IF NOT EXISTS reading_level_fre NUMERIC(4, 1);

ALTER TABLE course.module_assignments
    ADD COLUMN IF NOT EXISTS reading_level_fkgl NUMERIC(4, 1),
    ADD COLUMN IF NOT EXISTS reading_level_fre NUMERIC(4, 1);

CREATE TABLE IF NOT EXISTS i18n.simplified_content_cache (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    source_item_id UUID NOT NULL,
    source_item_type TEXT NOT NULL CHECK (source_item_type IN ('content_page', 'assignment')),
    target_fkgl INTEGER NOT NULL CHECK (target_fkgl >= 0 AND target_fkgl <= 12),
    simplified_text TEXT NOT NULL,
    computed_fkgl NUMERIC(4, 1),
    generated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (source_item_id, source_item_type, target_fkgl)
);

CREATE INDEX IF NOT EXISTS idx_simplified_cache_lookup
    ON i18n.simplified_content_cache (source_item_id, target_fkgl);

ALTER TABLE course.course_enrollments
    ADD COLUMN IF NOT EXISTS reading_level_override INTEGER CHECK (
        reading_level_override IS NULL
        OR (
            reading_level_override >= 0
            AND reading_level_override <= 12
        )
    );

COMMENT ON COLUMN course.course_enrollments.reading_level_override IS
    'Target Flesch-Kincaid grade level for simplified content (IEP accommodation); NULL = none.';

ALTER TABLE settings.platform_app_settings
    ADD COLUMN IF NOT EXISTS reading_level_enabled BOOLEAN;

COMMENT ON COLUMN settings.platform_app_settings.reading_level_enabled IS
    'When true, FKGL scoring, editor badges, and AI content simplification are available (plan 11.6).';
