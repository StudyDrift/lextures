ALTER TABLE course.course_structure_items
    DROP CONSTRAINT IF EXISTS course_structure_items_title_not_api_error;

DROP FUNCTION IF EXISTS course.title_is_api_error_payload(text);
