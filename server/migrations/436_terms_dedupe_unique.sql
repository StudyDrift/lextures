-- Collapse duplicate academic terms (same org + name) and enforce uniqueness.
-- Historical pollution came from integration tests re-seeding Fall/Spring/Summer
-- into the shared seed org without cleanup; Canvas import also needs a stable
-- one-term-per-name identity when assigning courses.

WITH ranked AS (
    SELECT
        id,
        org_id,
        ROW_NUMBER() OVER (
            PARTITION BY org_id, lower(btrim(name))
            ORDER BY created_at ASC, id ASC
        ) AS rn,
        FIRST_VALUE(id) OVER (
            PARTITION BY org_id, lower(btrim(name))
            ORDER BY created_at ASC, id ASC
        ) AS keep_id
    FROM tenant.terms
),
dupes AS (
    SELECT id AS dupe_id, keep_id
    FROM ranked
    WHERE rn > 1
)
UPDATE course.courses c
SET term_id = d.keep_id, updated_at = NOW()
FROM dupes d
WHERE c.term_id = d.dupe_id;

WITH ranked AS (
    SELECT
        id,
        ROW_NUMBER() OVER (
            PARTITION BY org_id, lower(btrim(name))
            ORDER BY created_at ASC, id ASC
        ) AS rn,
        FIRST_VALUE(id) OVER (
            PARTITION BY org_id, lower(btrim(name))
            ORDER BY created_at ASC, id ASC
        ) AS keep_id
    FROM tenant.terms
),
dupes AS (
    SELECT id AS dupe_id, keep_id
    FROM ranked
    WHERE rn > 1
)
UPDATE course.course_sections s
SET term_id = d.keep_id, updated_at = NOW()
FROM dupes d
WHERE s.term_id = d.dupe_id;

WITH ranked AS (
    SELECT
        id,
        ROW_NUMBER() OVER (
            PARTITION BY org_id, lower(btrim(name))
            ORDER BY created_at ASC, id ASC
        ) AS rn,
        FIRST_VALUE(id) OVER (
            PARTITION BY org_id, lower(btrim(name))
            ORDER BY created_at ASC, id ASC
        ) AS keep_id
    FROM tenant.terms
),
dupes AS (
    SELECT id AS dupe_id, keep_id
    FROM ranked
    WHERE rn > 1
)
UPDATE catalog.catalog_sections cs
SET term_id = d.keep_id, updated_at = NOW()
FROM dupes d
WHERE cs.term_id = d.dupe_id;

WITH ranked AS (
    SELECT
        id,
        ROW_NUMBER() OVER (
            PARTITION BY org_id, lower(btrim(name))
            ORDER BY created_at ASC, id ASC
        ) AS rn,
        FIRST_VALUE(id) OVER (
            PARTITION BY org_id, lower(btrim(name))
            ORDER BY created_at ASC, id ASC
        ) AS keep_id
    FROM tenant.terms
),
dupes AS (
    SELECT id AS dupe_id, keep_id
    FROM ranked
    WHERE rn > 1
)
UPDATE tenant.academic_calendar_events e
SET term_id = d.keep_id
FROM dupes d
WHERE e.term_id = d.dupe_id;

WITH ranked AS (
    SELECT
        id,
        ROW_NUMBER() OVER (
            PARTITION BY org_id, lower(btrim(name))
            ORDER BY created_at ASC, id ASC
        ) AS rn
    FROM tenant.terms
)
DELETE FROM tenant.terms t
USING ranked r
WHERE t.id = r.id
  AND r.rn > 1;

CREATE UNIQUE INDEX IF NOT EXISTS uq_terms_org_lower_name
    ON tenant.terms (org_id, lower(btrim(name)));

COMMENT ON INDEX tenant.uq_terms_org_lower_name IS
    'One academic term name per organization (case-insensitive).';
