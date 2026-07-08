ALTER TABLE settings.intro_course_items
    DROP CONSTRAINT IF EXISTS intro_course_items_grade_policy_check;

ALTER TABLE settings.intro_course_items
    DROP COLUMN IF EXISTS grade_policy;