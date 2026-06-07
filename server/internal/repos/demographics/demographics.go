// Package demographics provides data access for student demographic flags (plan 13.13).
package demographics

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const MinSubgroupSize = 10

// Row is one student_demographics record.
type Row struct {
	StudentID         uuid.UUID
	FreeLunch         *bool
	ReducedLunch      *bool
	EllStatus         *bool
	DisabilityStatus  *bool
	RaceEthnicityCode *string
	HomelessIndicator *bool
	MigrantIndicator  *bool
	DataSource        string
	LastVerifiedAt    *time.Time
	UpdatedAt         time.Time
}

// UpsertInput is the payload for creating or updating demographics.
type UpsertInput struct {
	FreeLunch         *bool
	ReducedLunch      *bool
	EllStatus         *bool
	DisabilityStatus  *bool
	RaceEthnicityCode *string
	HomelessIndicator *bool
	MigrantIndicator  *bool
	DataSource        string
}

// GetByStudentID returns demographics for a student, or nil if none exist.
func GetByStudentID(ctx context.Context, pool *pgxpool.Pool, studentID uuid.UUID) (*Row, error) {
	var r Row
	err := pool.QueryRow(ctx, `
SELECT student_id, free_lunch, reduced_lunch, ell_status, disability_status,
       race_ethnicity_code, homeless_indicator, migrant_indicator,
       data_source, last_verified_at, updated_at
FROM compliance.student_demographics
WHERE student_id = $1
`, studentID).Scan(
		&r.StudentID, &r.FreeLunch, &r.ReducedLunch, &r.EllStatus, &r.DisabilityStatus,
		&r.RaceEthnicityCode, &r.HomelessIndicator, &r.MigrantIndicator,
		&r.DataSource, &r.LastVerifiedAt, &r.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &r, nil
}

// Upsert creates or updates a student's demographic record.
func Upsert(ctx context.Context, pool *pgxpool.Pool, studentID uuid.UUID, in UpsertInput) (*Row, error) {
	source := in.DataSource
	if source == "" {
		source = "manual"
	}
	_, err := pool.Exec(ctx, `
INSERT INTO compliance.student_demographics (
    student_id, free_lunch, reduced_lunch, ell_status, disability_status,
    race_ethnicity_code, homeless_indicator, migrant_indicator,
    data_source, last_verified_at, updated_at
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, now(), now())
ON CONFLICT (student_id) DO UPDATE SET
    free_lunch = COALESCE(EXCLUDED.free_lunch, compliance.student_demographics.free_lunch),
    reduced_lunch = COALESCE(EXCLUDED.reduced_lunch, compliance.student_demographics.reduced_lunch),
    ell_status = COALESCE(EXCLUDED.ell_status, compliance.student_demographics.ell_status),
    disability_status = COALESCE(EXCLUDED.disability_status, compliance.student_demographics.disability_status),
    race_ethnicity_code = COALESCE(EXCLUDED.race_ethnicity_code, compliance.student_demographics.race_ethnicity_code),
    homeless_indicator = COALESCE(EXCLUDED.homeless_indicator, compliance.student_demographics.homeless_indicator),
    migrant_indicator = COALESCE(EXCLUDED.migrant_indicator, compliance.student_demographics.migrant_indicator),
    data_source = EXCLUDED.data_source,
    last_verified_at = now(),
    updated_at = now()
`, studentID, in.FreeLunch, in.ReducedLunch, in.EllStatus, in.DisabilityStatus,
		in.RaceEthnicityCode, in.HomelessIndicator, in.MigrantIndicator, source)
	if err != nil {
		return nil, err
	}
	return GetByStudentID(ctx, pool, studentID)
}

// CountRecentDisclosureViews returns how many demographic disclosure log entries
// the accessor wrote in the last window (for bulk-access alerting, AC-4).
func CountRecentDisclosureViews(ctx context.Context, pool *pgxpool.Pool, accessorID uuid.UUID, window time.Duration) (int, error) {
	since := time.Now().UTC().Add(-window)
	var n int
	err := pool.QueryRow(ctx, `
SELECT COUNT(*)::int
FROM compliance.ferpa_disclosure_log
WHERE accessor_id = $1
  AND data_type = 'demographics'
  AND logged_at >= $2
`, accessorID, since).Scan(&n)
	return n, err
}

// StudentInSchoolSubtree returns true when the student is enrolled in a course under orgUnitID or its descendants.
func StudentInSchoolSubtree(ctx context.Context, pool *pgxpool.Pool, studentID, orgUnitID uuid.UUID) (bool, error) {
	var ok bool
	err := pool.QueryRow(ctx, `
WITH RECURSIVE subtree AS (
    SELECT id FROM tenant.org_units WHERE id = $2
    UNION ALL
    SELECT ou.id FROM tenant.org_units ou
    INNER JOIN subtree s ON ou.parent_id = s.id
)
SELECT EXISTS (
    SELECT 1
    FROM course.course_enrollments ce
    INNER JOIN course.courses c ON c.id = ce.course_id
    WHERE ce.user_id = $1
      AND ce.role = 'student'
      AND c.org_unit_id IN (SELECT id FROM subtree)
)
`, studentID, orgUnitID).Scan(&ok)
	return ok, err
}
