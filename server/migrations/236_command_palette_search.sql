-- Full-text search indexes for command-palette course and content discovery.

ALTER TABLE course.courses
    ADD COLUMN IF NOT EXISTS search_vector tsvector
    GENERATED ALWAYS AS (
        to_tsvector(
            'english',
            coalesce(course_code, '') || ' ' || coalesce(title, '')
        )
    ) STORED;

CREATE INDEX IF NOT EXISTS idx_courses_search_vector_gin
    ON course.courses USING gin (search_vector);

ALTER TABLE course.course_structure_items
    ADD COLUMN IF NOT EXISTS search_vector tsvector
    GENERATED ALWAYS AS (
        CASE
            WHEN kind NOT IN ('module', 'heading') AND archived = false
                THEN to_tsvector('english', coalesce(title, ''))
            ELSE ''::tsvector
        END
    ) STORED;

CREATE INDEX IF NOT EXISTS idx_course_structure_items_search_vector_gin
    ON course.course_structure_items USING gin (search_vector);
