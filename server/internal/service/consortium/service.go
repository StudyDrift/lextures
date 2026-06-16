// Package consortium implements cross-institutional course sharing (plan 14.18).
package consortium

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	repoConsortium "github.com/lextures/lextures/server/internal/repos/consortium"
)

var (
	ErrAgreementNotActive = errors.New("no active consortium agreement")
	ErrCourseNotShareable = errors.New("course is not consortium-shareable")
	ErrAlreadyEnrolled    = errors.New("already enrolled")
)

// EnrollGuestStudent creates a cross-institutional enrollment for a guest-org student.
func EnrollGuestStudent(ctx context.Context, pool *pgxpool.Pool, courseID, userID, homeOrgID uuid.UUID) error {
	var hostOrgID uuid.UUID
	var shareable bool
	err := pool.QueryRow(ctx, `
SELECT org_id, consortium_shareable FROM course.courses WHERE id = $1
`, courseID).Scan(&hostOrgID, &shareable)
	if errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("course not found")
	}
	if err != nil {
		return err
	}
	if !shareable {
		return ErrCourseNotShareable
	}
	ok, err := repoConsortium.ActiveAgreementExists(ctx, pool, hostOrgID, homeOrgID)
	if err != nil {
		return err
	}
	if !ok {
		return ErrAgreementNotActive
	}
	var userOrgID uuid.UUID
	err = pool.QueryRow(ctx, `SELECT org_id FROM "user".users WHERE id = $1`, userID).Scan(&userOrgID)
	if err != nil {
		return err
	}
	if userOrgID != homeOrgID {
		return fmt.Errorf("user org does not match home org")
	}

	var exists bool
	err = pool.QueryRow(ctx, `
SELECT EXISTS (
  SELECT 1 FROM course.course_enrollments ce
  WHERE ce.course_id = $1 AND ce.user_id = $2 AND ce.active
)
`, courseID, userID).Scan(&exists)
	if err != nil {
		return err
	}
	if exists {
		return ErrAlreadyEnrolled
	}

	tag, err := pool.Exec(ctx, `
INSERT INTO course.course_enrollments (course_id, user_id, role, active, home_org_id)
VALUES ($1, $2, 'student', true, $3)
`, courseID, userID, homeOrgID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("enrollment not created")
	}
	RecordEnrollment()
	return nil
}
