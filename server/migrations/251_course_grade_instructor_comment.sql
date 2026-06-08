-- Instructor feedback comment on a gradebook cell (SpeedGrader / submission preview).
ALTER TABLE course.course_grades
    ADD COLUMN IF NOT EXISTS instructor_comment TEXT;

COMMENT ON COLUMN course.course_grades.instructor_comment IS
    'Optional instructor feedback shown with the grade when posted.';
