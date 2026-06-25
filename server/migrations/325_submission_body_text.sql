-- Online text-entry body for module assignment submissions (GA-M2).

ALTER TABLE course.module_assignment_submissions
    ADD COLUMN IF NOT EXISTS body_text TEXT NOT NULL DEFAULT '';

COMMENT ON COLUMN course.module_assignment_submissions.body_text IS
    'Typed submission body when the assignment allows online text entry (no file required).';