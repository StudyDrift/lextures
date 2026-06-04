-- Plan 13.8: Library catalog, reading log, and book-club features.

CREATE SCHEMA IF NOT EXISTS library;

CREATE TABLE IF NOT EXISTS library.books (
    id           UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id       UUID        NOT NULL REFERENCES tenant.organizations (id) ON DELETE CASCADE,
    title        TEXT        NOT NULL,
    author       TEXT,
    isbn         TEXT,
    cover_url    TEXT,
    lexile_level INTEGER,
    fp_band      TEXT,        -- Fountas & Pinnell band A-Z
    grade_band   TEXT,        -- K-2, 3-5, 6-8, 9-12, K-12
    summary      TEXT,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS library_books_org_idx       ON library.books (org_id);
CREATE INDEX IF NOT EXISTS library_books_lexile_idx    ON library.books (org_id, lexile_level) WHERE lexile_level IS NOT NULL;
CREATE INDEX IF NOT EXISTS library_books_grade_idx     ON library.books (org_id, grade_band) WHERE grade_band IS NOT NULL;

COMMENT ON TABLE library.books IS
    'Plan 13.8: School (org) library catalog. FERPA: book titles/authors are not PII.';
COMMENT ON COLUMN library.books.fp_band IS 'Fountas & Pinnell reading band (A-Z).';
COMMENT ON COLUMN library.books.grade_band IS 'Grade band, e.g. K-2, 3-5, 6-8, 9-12, K-12.';

CREATE TABLE IF NOT EXISTS library.reading_log_entries (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    student_id  UUID        NOT NULL REFERENCES "user".users (id) ON DELETE CASCADE,
    book_id     UUID        REFERENCES library.books (id) ON DELETE SET NULL,
    book_title  TEXT,        -- free-text if book not in catalog
    log_date    DATE        NOT NULL,
    pages_read  INTEGER,
    reflection  TEXT,
    logged_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS reading_log_student_idx  ON library.reading_log_entries (student_id, log_date DESC);
CREATE INDEX IF NOT EXISTS reading_log_book_idx     ON library.reading_log_entries (book_id) WHERE book_id IS NOT NULL;

COMMENT ON TABLE library.reading_log_entries IS
    'Plan 13.8: Student reading log. FERPA education record; visible to teacher and parent only.';

ALTER TABLE settings.platform_app_settings
    ADD COLUMN IF NOT EXISTS ff_library BOOLEAN;

COMMENT ON COLUMN settings.platform_app_settings.ff_library IS
    'Plan 13.8: Enables library catalog, reading log, and reading dashboard.';
