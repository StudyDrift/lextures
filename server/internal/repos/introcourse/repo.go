// Package introcourse provides persistence helpers for the canonical intro course (IC01).
package introcourse

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Row is a minimal course row for provisioning.
type Row struct {
	ID         uuid.UUID
	CourseCode string
	Title      string
}

// LookupIDByShortCode returns the course id for shortCode, or nil when absent.
func LookupIDByShortCode(ctx context.Context, pool *pgxpool.Pool, shortCode string) (*uuid.UUID, error) {
	var id uuid.UUID
	err := pool.QueryRow(ctx, `
SELECT id FROM course.courses WHERE short_code = $1
`, shortCode).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &id, nil
}

// LookupIDByShortCodeTx is the transactional variant used under the provision advisory lock.
func LookupIDByShortCodeTx(ctx context.Context, tx pgx.Tx, shortCode string) (*uuid.UUID, error) {
	var id uuid.UUID
	err := tx.QueryRow(ctx, `
SELECT id FROM course.courses WHERE short_code = $1
`, shortCode).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &id, nil
}

// EnsureSystemInstructor verifies the migration-seeded guide user exists.
func EnsureSystemInstructor(ctx context.Context, tx pgx.Tx, userID uuid.UUID) error {
	var ok bool
	err := tx.QueryRow(ctx, `
SELECT EXISTS (
    SELECT 1 FROM "user".users
    WHERE id = $1 AND account_type = 'system'
)
`, userID).Scan(&ok)
	if err != nil {
		return err
	}
	if !ok {
		return errors.New("intro course system instructor missing; run migrations")
	}
	return nil
}

// DefaultOrgID returns the default tenant org id.
func DefaultOrgID(ctx context.Context, tx pgx.Tx) (uuid.UUID, error) {
	var id uuid.UUID
	err := tx.QueryRow(ctx, `
SELECT id FROM tenant.organizations WHERE slug = 'default' LIMIT 1
`).Scan(&id)
	return id, err
}

// CreateCourse inserts the canonical intro course row.
func CreateCourse(ctx context.Context, tx pgx.Tx, orgID, createdBy uuid.UUID, now time.Time) (uuid.UUID, error) {
	var id uuid.UUID
	err := tx.QueryRow(ctx, `
INSERT INTO course.courses (
    course_code,
    short_code,
    title,
    description,
    course_type,
    created_by_user_id,
    org_id,
    published,
    visible_from,
    grading_scale
) VALUES ($1, $2, $3, $4, 'traditional', $5, $6, TRUE, $7, 'letter_plus_minus')
RETURNING id
`, CourseCode, ShortCode, Title, Description, createdBy, orgID, now).Scan(&id)
	return id, err
}

// ReconcileCourse updates canonical fields on an existing intro course.
func ReconcileCourse(ctx context.Context, tx pgx.Tx, courseID uuid.UUID, now time.Time) error {
	_, err := tx.Exec(ctx, `
UPDATE course.courses
SET
    title = $2,
    description = $3,
    published = TRUE,
    visible_from = COALESCE(visible_from, $4),
    starts_at = NULL,
    ends_at = NULL,
    hidden_at = NULL,
    grading_scale = 'letter_plus_minus',
    updated_at = NOW()
WHERE id = $1
`, courseID, Title, Description, now)
	return err
}

// EnsureTeacherEnrollment enrolls the system instructor as teacher with grants.
func EnsureTeacherEnrollment(ctx context.Context, tx pgx.Tx, courseID, teacherID uuid.UUID, courseCode string) error {
	if _, err := tx.Exec(ctx, `
INSERT INTO course.course_enrollments (course_id, user_id, role)
VALUES ($1, $2, 'teacher')
ON CONFLICT (course_id, user_id, role) DO NOTHING
`, courseID, teacherID); err != nil {
		return err
	}
	return seedTeacherGrants(ctx, tx, teacherID, courseID, courseCode)
}

func seedTeacherGrants(ctx context.Context, tx pgx.Tx, userID, courseID uuid.UUID, courseCode string) error {
	prefix := "course:" + courseCode + ":"
	perms := []string{
		prefix + "item:create",
		prefix + "items:create",
		prefix + "enrollments:read",
		prefix + "enrollments:update",
		prefix + "gradebook:view",
		prefix + "attendance:manage",
	}
	for _, perm := range perms {
		if _, err := tx.Exec(ctx, `
INSERT INTO course.user_course_grants (user_id, course_id, permission_string)
VALUES ($1, $2, $3)
ON CONFLICT (user_id, course_id, permission_string) DO NOTHING
`, userID, courseID, perm); err != nil {
			return err
		}
	}
	return nil
}

type assignmentGroupSpec struct {
	SortOrder     int
	Name          string
	WeightPercent float64
}

var defaultAssignmentGroups = []assignmentGroupSpec{
	{0, "Participation", 10},
	{1, "Quizzes", 50},
	{2, "Assignments", 40},
}

// IsIntroCourseID reports whether courseID is the canonical intro course.
func IsIntroCourseID(ctx context.Context, q interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}, courseID uuid.UUID) (bool, error) {
	var ok bool
	err := q.QueryRow(ctx, `
SELECT EXISTS (
    SELECT 1 FROM course.courses WHERE id = $1 AND short_code = $2
)
`, courseID, ShortCode).Scan(&ok)
	return ok, err
}

// EnsureAssignmentGroups seeds the default weighted groups (IC04 refines weights).
func EnsureAssignmentGroups(ctx context.Context, tx pgx.Tx, courseID uuid.UUID) error {
	for _, g := range defaultAssignmentGroups {
		if _, err := tx.Exec(ctx, `
INSERT INTO course.assignment_groups (course_id, sort_order, name, weight_percent)
VALUES ($1, $2, $3, $4)
ON CONFLICT (course_id, sort_order) DO UPDATE SET
    name = EXCLUDED.name,
    weight_percent = EXCLUDED.weight_percent,
    updated_at = NOW()
`, courseID, g.SortOrder, g.Name, g.WeightPercent); err != nil {
			return err
		}
	}
	return nil
}