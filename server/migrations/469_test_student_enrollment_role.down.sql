-- Revert test_student dual-enroll seats back to student, then drop the catalog role.

UPDATE course.course_enrollments
SET role = 'student'
WHERE role = 'test_student';

DELETE FROM course.enrollment_roles
WHERE role_key = 'test_student';
