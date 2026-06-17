package enrollment

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ViewerInvitationInfo describes a pending enrollment invitation for the viewer.
type ViewerInvitationInfo struct {
	Pending      bool
	EnrollmentID *uuid.UUID
}

// ViewerInvitationForCourse returns whether the viewer has a pending invitation in the course.
func ViewerInvitationForCourse(ctx context.Context, pool *pgxpool.Pool, courseCode string, userID uuid.UUID) (ViewerInvitationInfo, error) {
	var out ViewerInvitationInfo
	var eid uuid.UUID
	err := pool.QueryRow(ctx, `
SELECT ce.id
FROM course.course_enrollments ce
INNER JOIN course.courses c ON c.id = ce.course_id
WHERE c.course_code = $1
  AND ce.user_id = $2
  AND ce.invitation_pending
LIMIT 1
`, courseCode, userID).Scan(&eid)
	if errors.Is(err, pgx.ErrNoRows) {
		return out, nil
	}
	if err != nil {
		return out, err
	}
	out.Pending = true
	out.EnrollmentID = &eid
	return out, nil
}

// ApproveInvitation activates a pending enrollment owned by the viewer.
func ApproveInvitation(ctx context.Context, tx pgx.Tx, enrollmentID, userID uuid.UUID) (courseID uuid.UUID, courseCode string, err error) {
	err = tx.QueryRow(ctx, `
UPDATE course.course_enrollments ce
SET active = TRUE,
    invitation_pending = FALSE
FROM course.courses c
WHERE ce.id = $1
  AND ce.user_id = $2
  AND ce.invitation_pending
  AND c.id = ce.course_id
RETURNING ce.course_id, c.course_code
`, enrollmentID, userID).Scan(&courseID, &courseCode)
	if errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, "", ErrInvitationNotFound
	}
	return courseID, courseCode, err
}

// DeclineInvitation deletes a pending enrollment owned by the viewer.
func DeclineInvitation(ctx context.Context, tx pgx.Tx, enrollmentID, userID uuid.UUID) (courseID uuid.UUID, courseCode string, err error) {
	err = tx.QueryRow(ctx, `
DELETE FROM course.course_enrollments ce
USING course.courses c
WHERE ce.id = $1
  AND ce.user_id = $2
  AND ce.invitation_pending
  AND c.id = ce.course_id
RETURNING ce.course_id, c.course_code
`, enrollmentID, userID).Scan(&courseID, &courseCode)
	if errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, "", ErrInvitationNotFound
	}
	return courseID, courseCode, err
}

// ErrInvitationNotFound is returned when an invitation row is missing or not owned by the viewer.
var ErrInvitationNotFound = errors.New("enrollment invitation not found")