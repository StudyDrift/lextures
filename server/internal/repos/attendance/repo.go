// Package attendance persists plan 13.2 daily attendance records.
package attendance

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Code is one row from course.attendance_codes.
type Code struct {
	ID        uuid.UUID
	OrgID     uuid.UUID
	Code      string
	Label     string
	StateCode *string
	Category  string
}

// Record is one row from course.attendance_records joined with code info.
type Record struct {
	ID         uuid.UUID
	StudentID  uuid.UUID
	SectionID  uuid.UUID
	SchoolID   *uuid.UUID
	Date       time.Time
	Period     *string
	CodeID     uuid.UUID
	Code       string
	CodeLabel  string
	Category   string
	Note       *string
	RecordedBy *uuid.UUID
	RecordedAt time.Time
	UpdatedAt  time.Time
}

// StudentRow is a roster entry for a section.
type StudentRow struct {
	UserID      uuid.UUID
	DisplayName *string
	Email       string
}

// UpsertRow is one input row for BatchUpsert.
type UpsertRow struct {
	StudentID  uuid.UUID
	SectionID  uuid.UUID
	SchoolID   *uuid.UUID
	Date       time.Time
	Period     *string
	CodeID     uuid.UUID
	Note       *string
	RecordedBy *uuid.UUID
}

// DashboardEntry summarises one section's attendance for the dashboard.
type DashboardEntry struct {
	SectionID    uuid.UUID
	SectionCode  string
	CourseName   string
	Date         time.Time
	TotalStudents int
	PresentCount  int
	AbsentCount   int
	TardyCount    int
	NotTaken      bool
}

// ListCodes returns all attendance codes for an org ordered by code.
func ListCodes(ctx context.Context, pool *pgxpool.Pool, orgID uuid.UUID) ([]Code, error) {
	rows, err := pool.Query(ctx, `
SELECT id, org_id, code, label, state_code, category
FROM course.attendance_codes
WHERE org_id = $1
ORDER BY code ASC
`, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Code
	for rows.Next() {
		var c Code
		var sc *string
		if err := rows.Scan(&c.ID, &c.OrgID, &c.Code, &c.Label, &sc, &c.Category); err != nil {
			return nil, err
		}
		c.StateCode = sc
		out = append(out, c)
	}
	return out, rows.Err()
}

// UpsertCode creates or updates an attendance code for an org.
func UpsertCode(ctx context.Context, pool *pgxpool.Pool, orgID uuid.UUID, code, label string, stateCode *string, category string) (*Code, error) {
	row := pool.QueryRow(ctx, `
INSERT INTO course.attendance_codes (org_id, code, label, state_code, category)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (org_id, code) DO UPDATE
    SET label = EXCLUDED.label,
        state_code = EXCLUDED.state_code,
        category = EXCLUDED.category
RETURNING id, org_id, code, label, state_code, category
`, orgID, code, label, stateCode, category)
	var c Code
	var sc *string
	if err := row.Scan(&c.ID, &c.OrgID, &c.Code, &c.Label, &sc, &c.Category); err != nil {
		return nil, err
	}
	c.StateCode = sc
	return &c, nil
}

// DeleteCode removes an attendance code if no records reference it.
// Returns (true, nil) if deleted, (false, nil) if not found, (false, err) on error.
func DeleteCode(ctx context.Context, pool *pgxpool.Pool, orgID, codeID uuid.UUID) (bool, error) {
	var inUse bool
	if err := pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM course.attendance_records WHERE code_id = $1)`,
		codeID).Scan(&inUse); err != nil {
		return false, err
	}
	if inUse {
		return false, errors.New("code is referenced by attendance records")
	}
	tag, err := pool.Exec(ctx,
		`DELETE FROM course.attendance_codes WHERE id = $1 AND org_id = $2`,
		codeID, orgID)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}

// SeedDefaultCodes inserts the 5 standard codes for an org if none exist yet.
func SeedDefaultCodes(ctx context.Context, pool *pgxpool.Pool, orgID uuid.UUID) error {
	defaults := []struct {
		code, label, category string
	}{
		{"P", "Present", "present"},
		{"AU", "Absent-Unexcused", "absent"},
		{"AE", "Absent-Excused", "absent"},
		{"TU", "Tardy-Unexcused", "tardy"},
		{"TE", "Tardy-Excused", "tardy"},
	}
	for _, d := range defaults {
		_, err := pool.Exec(ctx, `
INSERT INTO course.attendance_codes (org_id, code, label, category)
VALUES ($1, $2, $3, $4)
ON CONFLICT (org_id, code) DO NOTHING
`, orgID, d.code, d.label, d.category)
		if err != nil {
			return err
		}
	}
	return nil
}

// ListRosterForSection returns students enrolled in a section.
func ListRosterForSection(ctx context.Context, pool *pgxpool.Pool, sectionID uuid.UUID) ([]StudentRow, error) {
	rows, err := pool.Query(ctx, `
SELECT ce.user_id, u.display_name, u.email
FROM course.course_enrollments ce
JOIN "user".users u ON u.id = ce.user_id
WHERE ce.section_id = $1 AND ce.active
  AND ce.role IN ('student', 'learner')
ORDER BY u.display_name ASC NULLS LAST, u.email ASC
`, sectionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []StudentRow
	for rows.Next() {
		var s StudentRow
		var dn sql.NullString
		if err := rows.Scan(&s.UserID, &dn, &s.Email); err != nil {
			return nil, err
		}
		if dn.Valid && dn.String != "" {
			s.DisplayName = &dn.String
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// ListForSection returns all attendance records for a section on a given date.
func ListForSection(ctx context.Context, pool *pgxpool.Pool, sectionID uuid.UUID, date time.Time) ([]Record, error) {
	rows, err := pool.Query(ctx, `
SELECT ar.id, ar.student_id, ar.section_id, ar.school_id, ar.date, ar.period,
       ar.code_id, ac.code, ac.label, ac.category, ar.note, ar.recorded_by, ar.recorded_at, ar.updated_at
FROM course.attendance_records ar
JOIN course.attendance_codes ac ON ac.id = ar.code_id
WHERE ar.section_id = $1 AND ar.date = $2::date
ORDER BY ar.updated_at DESC
`, sectionID, date.Format("2006-01-02"))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanRecords(rows)
}

// ListForStudent returns a student's attendance history ordered newest first.
func ListForStudent(ctx context.Context, pool *pgxpool.Pool, studentID uuid.UUID, limit int) ([]Record, error) {
	if limit <= 0 {
		limit = 200
	}
	rows, err := pool.Query(ctx, `
SELECT ar.id, ar.student_id, ar.section_id, ar.school_id, ar.date, ar.period,
       ar.code_id, ac.code, ac.label, ac.category, ar.note, ar.recorded_by, ar.recorded_at, ar.updated_at
FROM course.attendance_records ar
JOIN course.attendance_codes ac ON ac.id = ar.code_id
WHERE ar.student_id = $1
ORDER BY ar.date DESC, ar.period ASC NULLS LAST
LIMIT $2
`, studentID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanRecords(rows)
}

// BatchUpsert upserts a batch of attendance records in a single transaction.
func BatchUpsert(ctx context.Context, pool *pgxpool.Pool, rows []UpsertRow) error {
	if len(rows) == 0 {
		return nil
	}
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	for _, r := range rows {
		_, err := tx.Exec(ctx, `
INSERT INTO course.attendance_records
    (student_id, section_id, school_id, date, period, code_id, note, recorded_by, recorded_at, updated_at)
VALUES ($1, $2, $3, $4::date, $5, $6, $7, $8, now(), now())
ON CONFLICT (student_id, section_id, date, COALESCE(period, ''))
DO UPDATE SET
    code_id = EXCLUDED.code_id,
    note = EXCLUDED.note,
    recorded_by = EXCLUDED.recorded_by,
    updated_at = now()
`,
			r.StudentID, r.SectionID, r.SchoolID,
			r.Date.Format("2006-01-02"), r.Period,
			r.CodeID, r.Note, r.RecordedBy,
		)
		if err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

// IsWithinEditWindow returns true if the attendance date is within 5 calendar days of today.
func IsWithinEditWindow(date time.Time) bool {
	today := time.Now().UTC().Truncate(24 * time.Hour)
	cutoff := today.AddDate(0, 0, -5)
	d := date.UTC().Truncate(24 * time.Hour)
	return !d.Before(cutoff) && !d.After(today)
}

// DashboardForOrgUnit returns per-section attendance summary for a school org unit on a date.
func DashboardForOrgUnit(ctx context.Context, pool *pgxpool.Pool, orgUnitID uuid.UUID, date time.Time) ([]DashboardEntry, error) {
	rows, err := pool.Query(ctx, `
WITH sections AS (
    SELECT cs.id, cs.section_code, c.title AS course_name
    FROM course.course_sections cs
    JOIN course.courses c ON c.id = cs.course_id
    JOIN course.course_enrollments ce_teacher
        ON ce_teacher.section_id = cs.id AND ce_teacher.role IN ('teacher', 'instructor', 'owner') AND ce_teacher.active
    WHERE cs.status = 'active'
      AND c.org_unit_id = $1
),
roster AS (
    SELECT ce.section_id, COUNT(DISTINCT ce.user_id) AS total
    FROM course.course_enrollments ce
    JOIN sections s ON s.id = ce.section_id
    WHERE ce.active AND ce.role IN ('student', 'learner')
    GROUP BY ce.section_id
),
taken AS (
    SELECT ar.section_id,
           COUNT(*) FILTER (WHERE ac.category = 'present') AS present_count,
           COUNT(*) FILTER (WHERE ac.category = 'absent')  AS absent_count,
           COUNT(*) FILTER (WHERE ac.category = 'tardy')   AS tardy_count
    FROM course.attendance_records ar
    JOIN course.attendance_codes ac ON ac.id = ar.code_id
    JOIN sections s ON s.id = ar.section_id
    WHERE ar.date = $2::date
    GROUP BY ar.section_id
)
SELECT s.id, s.section_code, s.course_name,
       COALESCE(r.total, 0) AS total,
       COALESCE(t.present_count, 0),
       COALESCE(t.absent_count, 0),
       COALESCE(t.tardy_count, 0),
       (t.section_id IS NULL) AS not_taken
FROM sections s
LEFT JOIN roster r ON r.section_id = s.id
LEFT JOIN taken t ON t.section_id = s.id
ORDER BY s.section_code ASC
`, orgUnitID, date.Format("2006-01-02"))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	dateVal := date.UTC().Truncate(24 * time.Hour)
	var out []DashboardEntry
	for rows.Next() {
		var e DashboardEntry
		if err := rows.Scan(
			&e.SectionID, &e.SectionCode, &e.CourseName,
			&e.TotalStudents, &e.PresentCount, &e.AbsentCount, &e.TardyCount,
			&e.NotTaken,
		); err != nil {
			return nil, err
		}
		e.Date = dateVal
		out = append(out, e)
	}
	return out, rows.Err()
}

// InstructorForSection returns the instructor_user_id for a section (nil if none).
func InstructorForSection(ctx context.Context, pool *pgxpool.Pool, sectionID uuid.UUID) (*uuid.UUID, error) {
	var id *uuid.UUID
	row := pool.QueryRow(ctx,
		`SELECT instructor_user_id FROM course.course_sections WHERE id = $1`, sectionID)
	var s *string
	if err := row.Scan(&s); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	if s != nil {
		u, err := uuid.Parse(*s)
		if err != nil {
			return nil, err
		}
		id = &u
	}
	return id, nil
}

// OrgIDForSection returns the org_id of the organization that owns the section's course.
func OrgIDForSection(ctx context.Context, pool *pgxpool.Pool, sectionID uuid.UUID) (*uuid.UUID, error) {
	row := pool.QueryRow(ctx, `
SELECT c.org_id
FROM course.course_sections cs
JOIN course.courses c ON c.id = cs.course_id
WHERE cs.id = $1
`, sectionID)
	var orgID uuid.UUID
	if err := row.Scan(&orgID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &orgID, nil
}

func scanRecords(rows pgx.Rows) ([]Record, error) {
	var out []Record
	for rows.Next() {
		var r Record
		var schoolID sql.NullString
		var period sql.NullString
		var note sql.NullString
		var recordedBy sql.NullString
		if err := rows.Scan(
			&r.ID, &r.StudentID, &r.SectionID, &schoolID, &r.Date, &period,
			&r.CodeID, &r.Code, &r.CodeLabel, &r.Category, &note, &recordedBy,
			&r.RecordedAt, &r.UpdatedAt,
		); err != nil {
			return nil, err
		}
		if schoolID.Valid && schoolID.String != "" {
			u, err := uuid.Parse(schoolID.String)
			if err != nil {
				return nil, err
			}
			r.SchoolID = &u
		}
		if period.Valid {
			r.Period = &period.String
		}
		if note.Valid {
			r.Note = &note.String
		}
		if recordedBy.Valid && recordedBy.String != "" {
			u, err := uuid.Parse(recordedBy.String)
			if err != nil {
				return nil, err
			}
			r.RecordedBy = &u
		}
		out = append(out, r)
	}
	return out, rows.Err()
}
