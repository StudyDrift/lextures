-- IC04: per-item auto-grade policy for intro course curriculum sync.
ALTER TABLE settings.intro_course_items
    ADD COLUMN IF NOT EXISTS grade_policy TEXT NULL;

ALTER TABLE settings.intro_course_items DROP CONSTRAINT IF EXISTS intro_course_items_grade_policy_check;
ALTER TABLE settings.intro_course_items
    ADD CONSTRAINT intro_course_items_grade_policy_check CHECK (
        grade_policy IS NULL OR grade_policy IN ('quiz_autoscore', 'completion_full', 'grader_agent')
    );

COMMENT ON COLUMN settings.intro_course_items.grade_policy IS
    'Auto-grade policy for intro course items: quiz_autoscore, completion_full, or grader_agent (IC04).';