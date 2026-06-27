-- Plan 13.4: Per-course report cards feature flag (default off).

ALTER TABLE course.courses
    ADD COLUMN IF NOT EXISTS report_cards_enabled boolean NOT NULL DEFAULT false;

COMMENT ON COLUMN course.courses.report_cards_enabled IS
    'When true, instructors can author and release district-formatted report cards for this course.';