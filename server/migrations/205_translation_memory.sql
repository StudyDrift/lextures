-- Plan 11.5: Course content translation workflow, translation memory, and glossaries.

CREATE EXTENSION IF NOT EXISTS pg_trgm;

CREATE TABLE IF NOT EXISTS i18n.content_translations (
    id                        UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    source_item_id            UUID NOT NULL,
    source_item_type          TEXT NOT NULL CHECK (
        source_item_type IN ('content_page', 'assignment', 'quiz_question')
    ),
    source_locale             TEXT NOT NULL DEFAULT 'en',
    target_locale             TEXT NOT NULL,
    translated_title          TEXT,
    translated_body           TEXT,
    is_draft                  BOOLEAN NOT NULL DEFAULT TRUE,
    machine_translation_draft BOOLEAN NOT NULL DEFAULT FALSE,
    reviewed_by               UUID REFERENCES "user".users (id),
    published_at              TIMESTAMPTZ,
    version                   BIGINT NOT NULL DEFAULT 1,
    created_at                TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at                TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (source_item_id, source_item_type, target_locale)
);

CREATE INDEX IF NOT EXISTS idx_content_translations_item
    ON i18n.content_translations (source_item_id, source_item_type);

CREATE TABLE IF NOT EXISTS i18n.translation_memory (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    source_locale   TEXT NOT NULL,
    target_locale   TEXT NOT NULL,
    source_text     TEXT NOT NULL,
    source_hash     TEXT NOT NULL,
    translated_text TEXT NOT NULL,
    quality_score   NUMERIC(3, 2) NOT NULL DEFAULT 1.0,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (source_locale, target_locale, source_hash)
);

CREATE INDEX IF NOT EXISTS idx_tm_locale_pair
    ON i18n.translation_memory (source_locale, target_locale);

CREATE INDEX IF NOT EXISTS idx_tm_source_trgm
    ON i18n.translation_memory USING GIN (source_text gin_trgm_ops);

CREATE TABLE IF NOT EXISTS i18n.course_glossaries (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    course_id     UUID NOT NULL REFERENCES course.courses (id) ON DELETE CASCADE,
    source_locale TEXT NOT NULL,
    target_locale TEXT NOT NULL,
    source_term   TEXT NOT NULL,
    target_term   TEXT NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (course_id, source_locale, target_locale, source_term)
);

CREATE INDEX IF NOT EXISTS idx_course_glossaries_course
    ON i18n.course_glossaries (course_id, source_locale, target_locale);

ALTER TABLE course.course_enrollments
    ADD COLUMN IF NOT EXISTS content_locale TEXT;

COMMENT ON COLUMN course.course_enrollments.content_locale IS
    'Student preferred course content locale (BCP 47); NULL uses course/source default.';

ALTER TABLE settings.platform_app_settings
    ADD COLUMN IF NOT EXISTS translation_memory_enabled BOOLEAN;

COMMENT ON COLUMN settings.platform_app_settings.translation_memory_enabled IS
    'When true, course content translation editor, TM, glossary, and student locale selector are available (plan 11.5).';
