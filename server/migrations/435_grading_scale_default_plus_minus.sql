-- Default new courses to letter grades with plus/minus (A+, B-, …).

ALTER TABLE course.courses
    ALTER COLUMN grading_scale SET DEFAULT 'letter_plus_minus';
