package enrollment

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// GetCourseCreatorUserID returns courses.created_by_user_id for a course code.
func GetCourseCreatorUserID(ctx context.Context, pool *pgxpool.Pool, courseCode string) (*uuid.UUID, error) {
	if pool == nil {
		return nil, errors.New("db pool is nil")
	}
	var id uuid.UUID
	err := pool.QueryRow(ctx, `
SELECT created_by_user_id FROM course.courses WHERE course_code = $1
`, courseCode).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &id, nil
}

// DeleteEnrollmentsExceptCreatorTeacher removes every enrollment except the creator's teacher row.
func DeleteEnrollmentsExceptCreatorTeacher(ctx context.Context, pool *pgxpool.Pool, courseID, creatorUserID uuid.UUID) error {
	if pool == nil {
		return errors.New("db pool is nil")
	}
	_, err := pool.Exec(ctx, `
DELETE FROM course.course_enrollments
WHERE course_id = $1
  AND NOT (user_id = $2 AND role = 'teacher')
`, courseID, creatorUserID)
	return err
}

// InsertStudentIfMissing inserts a student enrollment when missing.
func InsertStudentIfMissing(ctx context.Context, pool *pgxpool.Pool, courseID, userID uuid.UUID) error {
	if pool == nil {
		return errors.New("db pool is nil")
	}
	_, err := pool.Exec(ctx, `
INSERT INTO course.course_enrollments (course_id, user_id, role)
VALUES ($1, $2, 'student')
ON CONFLICT (course_id, user_id, role) DO NOTHING
`, courseID, userID)
	return err
}

// UpsertInstructorEnrollment ensures an instructor enrollment row exists.
func UpsertInstructorEnrollment(ctx context.Context, pool *pgxpool.Pool, courseID, userID uuid.UUID) error {
	if pool == nil {
		return errors.New("db pool is nil")
	}
	_, err := pool.Exec(ctx, `
INSERT INTO course.course_enrollments (course_id, user_id, role)
VALUES ($1, $2, 'instructor')
ON CONFLICT (course_id, user_id, role) DO NOTHING
`, courseID, userID)
	return err
}

// EnsureTeacherEnrollment ensures a teacher enrollment row exists.
func EnsureTeacherEnrollment(ctx context.Context, pool *pgxpool.Pool, courseID, userID uuid.UUID) error {
	if pool == nil {
		return errors.New("db pool is nil")
	}
	_, err := pool.Exec(ctx, `
INSERT INTO course.course_enrollments (course_id, user_id, role)
VALUES ($1, $2, 'teacher')
ON CONFLICT (course_id, user_id, role) DO NOTHING
`, courseID, userID)
	return err
}

// InsertEnrollmentRoleIfMissing inserts any enrollment role when missing (e.g. ta).
func InsertEnrollmentRoleIfMissing(ctx context.Context, pool *pgxpool.Pool, courseID, userID uuid.UUID, role string) error {
	if pool == nil {
		return errors.New("db pool is nil")
	}
	_, err := pool.Exec(ctx, `
INSERT INTO course.course_enrollments (course_id, user_id, role)
VALUES ($1, $2, $3)
ON CONFLICT (course_id, user_id, role) DO NOTHING
`, courseID, userID, role)
	return err
}
