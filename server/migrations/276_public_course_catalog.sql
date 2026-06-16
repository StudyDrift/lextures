-- Public course catalog & search (plan 15.1).
-- Adds discoverability fields to course.courses and the platform feature flag.

ALTER TABLE course.courses
    ADD COLUMN IF NOT EXISTS is_public         BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS catalog_category  TEXT,
    ADD COLUMN IF NOT EXISTS difficulty_level  TEXT
        CHECK (difficulty_level IN ('beginner', 'intermediate', 'advanced')),
    ADD COLUMN IF NOT EXISTS catalog_language  TEXT NOT NULL DEFAULT 'en',
    ADD COLUMN IF NOT EXISTS price_cents       INT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS enrollment_count  INT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS average_rating    NUMERIC(3, 2),
    ADD COLUMN IF NOT EXISTS catalog_slug      TEXT;

COMMENT ON COLUMN course.courses.is_public IS
    'When true, the course is listed in the public catalog (plan 15.1). Draft/private courses excluded.';
COMMENT ON COLUMN course.courses.catalog_category IS
    'Free-text catalog category used for browse filters (plan 15.1).';
COMMENT ON COLUMN course.courses.difficulty_level IS
    'beginner | intermediate | advanced; powers the level filter (plan 15.1).';
COMMENT ON COLUMN course.courses.catalog_language IS
    'BCP-47-ish language code for the catalog language filter (plan 15.1).';
COMMENT ON COLUMN course.courses.price_cents IS
    'Listed price in cents; 0 means free. Enrollment/payment gated by 15.3 (plan 15.1).';
COMMENT ON COLUMN course.courses.enrollment_count IS
    'Aggregate enrollment count surfaced on catalog cards; no PII (plan 15.1).';
COMMENT ON COLUMN course.courses.average_rating IS
    'Average learner rating (0-5) populated by 15.7; NULL when unrated (plan 15.1).';
COMMENT ON COLUMN course.courses.catalog_slug IS
    'URL-safe slug for public course landing pages (plan 15.1). Unique among public courses.';

-- Backfill deterministic unique slugs from the already-unique course_code.
UPDATE course.courses
SET catalog_slug = lower(regexp_replace(course_code, '[^a-zA-Z0-9]+', '-', 'g'))
WHERE catalog_slug IS NULL;

-- Composite browse index limited to public rows keeps the public catalog query cheap.
CREATE INDEX IF NOT EXISTS idx_courses_catalog
    ON course.courses (is_public, catalog_category, difficulty_level)
    WHERE is_public = TRUE;

-- Full-text search index restricted to public rows. course.courses.search_vector
-- already exists (migration 236); this partial GIN index serves the public catalog.
CREATE INDEX IF NOT EXISTS idx_courses_catalog_search
    ON course.courses USING gin (search_vector)
    WHERE is_public = TRUE;

-- Slugs must be unique among public courses so landing-page lookups are unambiguous.
CREATE UNIQUE INDEX IF NOT EXISTS idx_courses_catalog_slug
    ON course.courses (catalog_slug)
    WHERE is_public = TRUE AND catalog_slug IS NOT NULL;

ALTER TABLE settings.platform_app_settings
    ADD COLUMN IF NOT EXISTS ff_public_catalog BOOLEAN;

COMMENT ON COLUMN settings.platform_app_settings.ff_public_catalog IS
    'Enables the public, unauthenticated course catalog and search (plan 15.1).';
