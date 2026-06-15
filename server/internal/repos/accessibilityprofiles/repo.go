// Package accessibilityprofiles stores accessibility-office accommodation profiles
// (plan 14.16). Profiles are a coordinator-facing intake layer that propagates to the
// 2.11 course.student_accommodations override engine.
package accessibilityprofiles

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Profile is one accessibility.accommodation_profiles row.
type Profile struct {
	ID             uuid.UUID       `json:"id"`
	StudentID      uuid.UUID       `json:"studentId"`
	OrgID          uuid.UUID       `json:"orgId"`
	Accommodations []string        `json:"accommodations"`
	CustomParams   json.RawMessage `json:"customParams"`
	EffectiveFrom  time.Time       `json:"effectiveFrom"`
	EffectiveUntil *time.Time      `json:"effectiveUntil,omitempty"`
	AppliedID      *uuid.UUID      `json:"-"`
	NotifiedAt     *time.Time      `json:"notifiedAt,omitempty"`
	CreatedBy      uuid.UUID       `json:"createdBy"`
	IsActive       bool            `json:"isActive"`
	CreatedAt      time.Time       `json:"createdAt"`
	UpdatedAt      time.Time       `json:"updatedAt"`
}

const selectCols = `
    id, student_id, org_id, accommodations::text[], custom_params,
    effective_from, effective_until, applied_accommodation_id, notified_at,
    created_by, is_active, created_at, updated_at`

func scanProfile(row pgx.Row) (*Profile, error) {
	var p Profile
	var params []byte
	if err := row.Scan(
		&p.ID, &p.StudentID, &p.OrgID, &p.Accommodations, &params,
		&p.EffectiveFrom, &p.EffectiveUntil, &p.AppliedID, &p.NotifiedAt,
		&p.CreatedBy, &p.IsActive, &p.CreatedAt, &p.UpdatedAt,
	); err != nil {
		return nil, err
	}
	if len(params) == 0 {
		params = []byte(`{}`)
	}
	p.CustomParams = json.RawMessage(params)
	return &p, nil
}

// CreateInput is the payload for a new profile.
type CreateInput struct {
	StudentID      uuid.UUID
	OrgID          uuid.UUID
	Accommodations []string
	CustomParams   json.RawMessage
	EffectiveFrom  *time.Time
	EffectiveUntil *time.Time
	CreatedBy      uuid.UUID
}

// Create inserts a new active accommodation profile.
func Create(ctx context.Context, pool *pgxpool.Pool, in CreateInput) (*Profile, error) {
	params := in.CustomParams
	if len(params) == 0 {
		params = json.RawMessage(`{}`)
	}
	const q = `
INSERT INTO accessibility.accommodation_profiles
    (student_id, org_id, accommodations, custom_params, effective_from, effective_until, created_by)
VALUES ($1, $2, $3::text[]::accessibility.accommodation_type[], $4::jsonb,
        COALESCE($5::date, CURRENT_DATE), $6::date, $7)
RETURNING` + selectCols
	row := pool.QueryRow(ctx, q,
		in.StudentID, in.OrgID, in.Accommodations, []byte(params),
		in.EffectiveFrom, in.EffectiveUntil, in.CreatedBy,
	)
	return scanProfile(row)
}

// Get returns a single profile by id, or (nil, nil) if not found.
func Get(ctx context.Context, pool *pgxpool.Pool, id uuid.UUID) (*Profile, error) {
	p, err := scanProfile(pool.QueryRow(ctx, `SELECT`+selectCols+`
FROM accessibility.accommodation_profiles WHERE id = $1`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return p, err
}

// ListForOrg returns all profiles in an org, newest first.
func ListForOrg(ctx context.Context, pool *pgxpool.Pool, orgID uuid.UUID) ([]Profile, error) {
	rows, err := pool.Query(ctx, `SELECT`+selectCols+`
FROM accessibility.accommodation_profiles WHERE org_id = $1
ORDER BY is_active DESC, created_at DESC`, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return collect(rows)
}

// ListActiveForStudent returns the student's own active profiles, newest first.
func ListActiveForStudent(ctx context.Context, pool *pgxpool.Pool, studentID uuid.UUID) ([]Profile, error) {
	rows, err := pool.Query(ctx, `SELECT`+selectCols+`
FROM accessibility.accommodation_profiles
WHERE student_id = $1 AND is_active = TRUE
ORDER BY created_at DESC`, studentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return collect(rows)
}

func collect(rows pgx.Rows) ([]Profile, error) {
	var out []Profile
	for rows.Next() {
		p, err := scanProfile(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *p)
	}
	return out, rows.Err()
}

// UpdatePatch carries optional updates for a profile.
type UpdatePatch struct {
	Accommodations *[]string
	CustomParams   json.RawMessage
	EffectiveFrom  *time.Time
	EffectiveUntil *time.Time
	IsActive       *bool
}

// Update applies a partial patch and returns the updated profile.
func Update(ctx context.Context, pool *pgxpool.Pool, id uuid.UUID, patch UpdatePatch) (*Profile, error) {
	const q = `
UPDATE accessibility.accommodation_profiles SET
    accommodations  = COALESCE($2::text[]::accessibility.accommodation_type[], accommodations),
    custom_params   = COALESCE($3::jsonb, custom_params),
    effective_from  = COALESCE($4::date, effective_from),
    effective_until = CASE WHEN $5::boolean THEN $6::date ELSE effective_until END,
    is_active       = COALESCE($7::boolean, is_active),
    updated_at      = NOW()
WHERE id = $1
RETURNING` + selectCols
	var params []byte
	if len(patch.CustomParams) > 0 {
		params = []byte(patch.CustomParams)
	}
	row := pool.QueryRow(ctx, q,
		id, patch.Accommodations, params, patch.EffectiveFrom,
		patch.EffectiveUntil != nil, patch.EffectiveUntil,
		patch.IsActive,
	)
	p, err := scanProfile(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return p, err
}

// SetApplied records the propagated student_accommodations row id on the profile.
func SetApplied(ctx context.Context, pool *pgxpool.Pool, id uuid.UUID, appliedID *uuid.UUID) error {
	_, err := pool.Exec(ctx, `
UPDATE accessibility.accommodation_profiles
SET applied_accommodation_id = $2, updated_at = NOW()
WHERE id = $1`, id, appliedID)
	return err
}

// MarkNotified stamps notified_at = NOW() on the profile.
func MarkNotified(ctx context.Context, pool *pgxpool.Pool, id uuid.UUID) error {
	_, err := pool.Exec(ctx, `
UPDATE accessibility.accommodation_profiles
SET notified_at = NOW(), updated_at = NOW()
WHERE id = $1`, id)
	return err
}

// AffectedCourse is a course a student is enrolled in (FR-5 / notification targeting).
type AffectedCourse struct {
	CourseID   uuid.UUID
	CourseCode string
	Title      string
}

// CoursesForStudent lists the active-enrollment courses for a student (used to show
// "courses affected" and to find instructors to notify).
func CoursesForStudent(ctx context.Context, pool *pgxpool.Pool, studentID uuid.UUID) ([]AffectedCourse, error) {
	rows, err := pool.Query(ctx, `
SELECT DISTINCT c.id, c.course_code, c.title
FROM course.course_enrollments e
JOIN course.courses c ON c.id = e.course_id
WHERE e.user_id = $1
  AND e.role = 'student'
  AND e.active
ORDER BY c.course_code`, studentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []AffectedCourse
	for rows.Next() {
		var a AffectedCourse
		if err := rows.Scan(&a.CourseID, &a.CourseCode, &a.Title); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

// InstructorIDsForStudent returns distinct teaching-staff user ids across the
// student's enrolled courses (FR-4 notification recipients).
func InstructorIDsForStudent(ctx context.Context, pool *pgxpool.Pool, studentID uuid.UUID) ([]uuid.UUID, error) {
	rows, err := pool.Query(ctx, `
SELECT DISTINCT staff.user_id
FROM course.course_enrollments stu
JOIN course.course_enrollments staff ON staff.course_id = stu.course_id
WHERE stu.user_id = $1
  AND stu.role = 'student'
  AND stu.active
  AND staff.role IN ('owner', 'teacher', 'instructor')
  AND staff.active`, studentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}
