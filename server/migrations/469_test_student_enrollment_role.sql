-- Test Student enrollment role for staff "preview as student" (same-user dual enrollment).
-- Role is student-equivalent so gradebook, My Grades, and learner flows include the seat.
-- Display name in roster/gradebook is overridden to "Test Student" in application code.

INSERT INTO course.enrollment_roles (
    role_key,
    display_name,
    sort_order,
    is_staff,
    is_student_equivalent,
    can_grade,
    can_author_content
)
VALUES (
    'test_student',
    'Test Student',
    25,
    false,
    true,
    false,
    false
)
ON CONFLICT (role_key) DO UPDATE
SET display_name           = EXCLUDED.display_name,
    sort_order             = EXCLUDED.sort_order,
    is_staff               = false,
    is_student_equivalent  = true,
    can_grade              = false,
    can_author_content     = false;

COMMENT ON TABLE course.enrollment_roles IS
    'Labels and UI sort order for values of course.course_enrollments.role. test_student is staff preview seat.';

-- Convert existing staff dual-enroll "student" seats to test_student so roster/gradebook
-- show the Test Student label instead of the instructor's real name.
UPDATE course.course_enrollments ce
SET role = 'test_student'
WHERE ce.role = 'student'
  AND EXISTS (
    SELECT 1
    FROM course.course_enrollments staff
    INNER JOIN course.enrollment_roles er
        ON er.role_key = staff.role AND er.is_staff = true
    WHERE staff.course_id = ce.course_id
      AND staff.user_id = ce.user_id
      AND staff.active = true
  );
