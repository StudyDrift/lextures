-- Issue #676 — a failed course-structure reorder was stored as a page title.
-- Drop those rows, drop empty archived copies of the AI Essentials workflow page
-- on C-HUPCNF, and refuse any future title that is an API error envelope.

CREATE OR REPLACE FUNCTION course.title_is_api_error_payload(t text)
RETURNS boolean
LANGUAGE plpgsql
STABLE
AS $$
DECLARE
    parsed jsonb;
    trimmed text;
BEGIN
    IF t IS NULL THEN
        RETURN false;
    END IF;
    trimmed := btrim(t);
    IF left(trimmed, 1) <> '{' THEN
        RETURN false;
    END IF;
    BEGIN
        parsed := trimmed::jsonb;
    EXCEPTION WHEN others THEN
        RETURN false;
    END;
    RETURN jsonb_typeof(parsed -> 'error') = 'object'
        AND COALESCE(btrim(parsed -> 'error' ->> 'code'), '') <> ''
        AND COALESCE(btrim(parsed -> 'error' ->> 'message'), '') <> '';
END;
$$;

-- Empty archived duplicates left behind while moving
-- "What Makes a Strong AI-Augmented Workflow" on AI Essentials.
-- The live page in Module 7 keeps its body and is not archived.
DELETE FROM course.course_structure_items AS csi
USING course.courses AS c
WHERE csi.course_id = c.id
  AND c.course_code = 'C-HUPCNF'
  AND csi.archived = TRUE
  AND csi.kind = 'content_page'
  AND csi.title = 'What Makes a Strong AI-Augmented Workflow'
  AND NOT EXISTS (
      SELECT 1
      FROM course.module_content_pages AS p
      WHERE p.structure_item_id = csi.id
        AND length(btrim(p.markdown)) > 0
  );

DELETE FROM course.course_structure_items
WHERE course.title_is_api_error_payload(title);

ALTER TABLE course.course_structure_items
    DROP CONSTRAINT IF EXISTS course_structure_items_title_not_api_error;

ALTER TABLE course.course_structure_items
    ADD CONSTRAINT course_structure_items_title_not_api_error
        CHECK (NOT course.title_is_api_error_payload(title));
