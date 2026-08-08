package enrollment

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// RoleTestStudent is the catalog role_key for staff preview-as-learner seats.
const RoleTestStudent = "test_student"

// DisplayNameTestStudent is the roster/gradebook label for test_student enrollments.
const DisplayNameTestStudent = "Test Student"

// TestStudentDisplayName returns the public display label for an enrollment role.
// test_student rows always surface as "Test Student" rather than the user's real name.
func TestStudentDisplayName(role string, fallback string) string {
	if role == RoleTestStudent {
		return DisplayNameTestStudent
	}
	return fallback
}

// IsTestStudentRole reports whether role is the Test Student catalog key.
func IsTestStudentRole(role string) bool {
	return role == RoleTestStudent
}

// EnsureTestStudentEnrollment inserts an active test_student enrollment for userID in courseID
// if one does not already exist. Idempotent. Does not refresh course grants — caller must.
// Returns the enrollment id and whether a new row was created.
func EnsureTestStudentEnrollment(ctx context.Context, tx pgx.Tx, courseID, userID uuid.UUID) (enrollmentID uuid.UUID, created bool, err error) {
	err = tx.QueryRow(ctx, `
SELECT id
FROM course.course_enrollments
WHERE course_id = $1 AND user_id = $2 AND role = $3
LIMIT 1
`, courseID, userID, RoleTestStudent).Scan(&enrollmentID)
	if err == nil {
		// Reactivate if previously deactivated (leave lifecycle state alone when already active).
		_, err = tx.Exec(ctx, `
UPDATE course.course_enrollments
SET active = true,
    invitation_pending = false
WHERE id = $1 AND (active = false OR invitation_pending = true)
`, enrollmentID)
		if err != nil {
			return uuid.Nil, false, err
		}
		return enrollmentID, false, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, false, err
	}

	err = tx.QueryRow(ctx, `
INSERT INTO course.course_enrollments (course_id, user_id, role, active, invitation_pending)
VALUES ($1, $2, $3, true, false)
ON CONFLICT (course_id, user_id, role) DO UPDATE
SET active = true,
    invitation_pending = false
RETURNING id
`, courseID, userID, RoleTestStudent).Scan(&enrollmentID)
	if err != nil {
		return uuid.Nil, false, err
	}
	return enrollmentID, true, nil
}

// EnsureTestStudentEnrollmentPool is EnsureTestStudentEnrollment for a pool (begins its own tx).
// Prefer the tx form when refreshing grants in the same transaction.
func EnsureTestStudentEnrollmentPool(ctx context.Context, pool *pgxpool.Pool, courseID, userID uuid.UUID) (enrollmentID uuid.UUID, created bool, err error) {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return uuid.Nil, false, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	id, created, err := EnsureTestStudentEnrollment(ctx, tx, courseID, userID)
	if err != nil {
		return uuid.Nil, false, err
	}
	if err := tx.Commit(ctx); err != nil {
		return uuid.Nil, false, err
	}
	return id, created, nil
}
