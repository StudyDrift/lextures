// Package incompletegrades persists HE Incomplete grade records (plan 14.4).
package incompletegrades

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Status is the lifecycle of an incomplete grade record.
type Status string

const (
	StatusOpen     Status = "open"
	StatusResolved Status = "resolved"
	StatusLapsed   Status = "lapsed"
)

// Record is one incomplete_grade_records row with joined context for reports.
type Record struct {
	ID                 uuid.UUID
	EnrollmentID       uuid.UUID
	GrantedBy          uuid.UUID
	ExtensionDeadline  time.Time
	OutstandingItemIDs []uuid.UUID
	Notes              *string
	Status             Status
	ResolvedGrade      *string
	ResolvedAt         *time.Time
	ResolvedBy         *uuid.UUID
	Reminder30dSentAt  *time.Time
	Reminder7dSentAt   *time.Time
	Reminder1dSentAt   *time.Time
	CreatedAt          time.Time
}

// ReportRow extends Record with student/course context for the registrar report.
type ReportRow struct {
	Record
	StudentUserID     uuid.UUID
	StudentName       string
	CourseID          uuid.UUID
	CourseCode        string
	CourseTitle       string
	TermID            *uuid.UUID
	InstructorIDs     []uuid.UUID
	OutstandingTitles []string
}

// ReminderCandidate is an open incomplete due for a deadline reminder.
type ReminderCandidate struct {
	Record
	StudentUserID uuid.UUID
	StudentName   string
	CourseCode    string
	CourseTitle   string
	InstructorIDs []uuid.UUID
	DaysRemaining int
	ReminderKind  string // "30d", "7d", "1d"
}

// GetByEnrollmentID returns the incomplete record for an enrollment, if any.
func GetByEnrollmentID(ctx context.Context, pool *pgxpool.Pool, enrollmentID uuid.UUID) (*Record, error) {
	var row Record
	var status string
	var notes, resolvedGrade *string
	var outstanding []uuid.UUID
	err := pool.QueryRow(ctx, `
SELECT id, enrollment_id, granted_by, extension_deadline, outstanding_item_ids, notes,
       status, resolved_grade, resolved_at, resolved_by,
       reminder_30d_sent_at, reminder_7d_sent_at, reminder_1d_sent_at, created_at
FROM course.incomplete_grade_records
WHERE enrollment_id = $1
`, enrollmentID).Scan(
		&row.ID, &row.EnrollmentID, &row.GrantedBy, &row.ExtensionDeadline, &outstanding, &notes,
		&status, &resolvedGrade, &row.ResolvedAt, &row.ResolvedBy,
		&row.Reminder30dSentAt, &row.Reminder7dSentAt, &row.Reminder1dSentAt, &row.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	row.OutstandingItemIDs = outstanding
	row.Notes = notes
	row.Status = Status(status)
	row.ResolvedGrade = resolvedGrade
	return &row, nil
}

// GetOpenByEnrollmentIDs returns open records keyed by enrollment id.
func GetOpenByEnrollmentIDs(ctx context.Context, pool *pgxpool.Pool, enrollmentIDs []uuid.UUID) (map[uuid.UUID]Record, error) {
	out := make(map[uuid.UUID]Record)
	if len(enrollmentIDs) == 0 {
		return out, nil
	}
	rows, err := pool.Query(ctx, `
SELECT id, enrollment_id, granted_by, extension_deadline, outstanding_item_ids, notes,
       status, resolved_grade, resolved_at, resolved_by,
       reminder_30d_sent_at, reminder_7d_sent_at, reminder_1d_sent_at, created_at
FROM course.incomplete_grade_records
WHERE enrollment_id = ANY($1::uuid[]) AND status = 'open'
`, enrollmentIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var row Record
		var status string
		var notes, resolvedGrade *string
		var outstanding []uuid.UUID
		if err := rows.Scan(
			&row.ID, &row.EnrollmentID, &row.GrantedBy, &row.ExtensionDeadline, &outstanding, &notes,
			&status, &resolvedGrade, &row.ResolvedAt, &row.ResolvedBy,
			&row.Reminder30dSentAt, &row.Reminder7dSentAt, &row.Reminder1dSentAt, &row.CreatedAt,
		); err != nil {
			return nil, err
		}
		row.OutstandingItemIDs = outstanding
		row.Notes = notes
		row.Status = Status(status)
		row.ResolvedGrade = resolvedGrade
		out[row.EnrollmentID] = row
	}
	return out, rows.Err()
}

// InsertParams is input for creating a new incomplete record.
type InsertParams struct {
	EnrollmentID       uuid.UUID
	GrantedBy          uuid.UUID
	ExtensionDeadline  time.Time
	OutstandingItemIDs []uuid.UUID
	Notes              *string
}

// Reopen resets a prior record to open with new grant details.
func Reopen(ctx context.Context, pool *pgxpool.Pool, recordID uuid.UUID, p InsertParams) (*Record, error) {
	if len(p.OutstandingItemIDs) == 0 {
		p.OutstandingItemIDs = []uuid.UUID{}
	}
	var row Record
	var status string
	var notes, resolvedGrade *string
	var outstanding []uuid.UUID
	err := pool.QueryRow(ctx, `
UPDATE course.incomplete_grade_records
SET granted_by = $2,
    extension_deadline = $3,
    outstanding_item_ids = $4,
    notes = $5,
    status = 'open',
    resolved_grade = NULL,
    resolved_at = NULL,
    resolved_by = NULL,
    reminder_30d_sent_at = NULL,
    reminder_7d_sent_at = NULL,
    reminder_1d_sent_at = NULL
WHERE id = $1
RETURNING id, enrollment_id, granted_by, extension_deadline, outstanding_item_ids, notes,
          status, resolved_grade, resolved_at, resolved_by,
          reminder_30d_sent_at, reminder_7d_sent_at, reminder_1d_sent_at, created_at
`, recordID, p.GrantedBy, p.ExtensionDeadline, p.OutstandingItemIDs, p.Notes).Scan(
		&row.ID, &row.EnrollmentID, &row.GrantedBy, &row.ExtensionDeadline, &outstanding, &notes,
		&status, &resolvedGrade, &row.ResolvedAt, &row.ResolvedBy,
		&row.Reminder30dSentAt, &row.Reminder7dSentAt, &row.Reminder1dSentAt, &row.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	row.OutstandingItemIDs = outstanding
	row.Notes = notes
	row.Status = Status(status)
	row.ResolvedGrade = resolvedGrade
	return &row, nil
}

// Insert creates a new open incomplete record.
func Insert(ctx context.Context, pool *pgxpool.Pool, p InsertParams) (*Record, error) {
	if len(p.OutstandingItemIDs) == 0 {
		p.OutstandingItemIDs = []uuid.UUID{}
	}
	var row Record
	var status string
	var notes, resolvedGrade *string
	var outstanding []uuid.UUID
	err := pool.QueryRow(ctx, `
INSERT INTO course.incomplete_grade_records
    (enrollment_id, granted_by, extension_deadline, outstanding_item_ids, notes)
VALUES ($1, $2, $3, $4, $5)
RETURNING id, enrollment_id, granted_by, extension_deadline, outstanding_item_ids, notes,
          status, resolved_grade, resolved_at, resolved_by,
          reminder_30d_sent_at, reminder_7d_sent_at, reminder_1d_sent_at, created_at
`, p.EnrollmentID, p.GrantedBy, p.ExtensionDeadline, p.OutstandingItemIDs, p.Notes).Scan(
		&row.ID, &row.EnrollmentID, &row.GrantedBy, &row.ExtensionDeadline, &outstanding, &notes,
		&status, &resolvedGrade, &row.ResolvedAt, &row.ResolvedBy,
		&row.Reminder30dSentAt, &row.Reminder7dSentAt, &row.Reminder1dSentAt, &row.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	row.OutstandingItemIDs = outstanding
	row.Notes = notes
	row.Status = Status(status)
	row.ResolvedGrade = resolvedGrade
	return &row, nil
}

// UpdateExtensionDeadline updates the deadline on an open record.
func UpdateExtensionDeadline(ctx context.Context, pool *pgxpool.Pool, recordID uuid.UUID, deadline time.Time) (*Record, error) {
	var row Record
	var status string
	var notes, resolvedGrade *string
	var outstanding []uuid.UUID
	err := pool.QueryRow(ctx, `
UPDATE course.incomplete_grade_records
SET extension_deadline = $2
WHERE id = $1 AND status = 'open'
RETURNING id, enrollment_id, granted_by, extension_deadline, outstanding_item_ids, notes,
          status, resolved_grade, resolved_at, resolved_by,
          reminder_30d_sent_at, reminder_7d_sent_at, reminder_1d_sent_at, created_at
`, recordID, deadline).Scan(
		&row.ID, &row.EnrollmentID, &row.GrantedBy, &row.ExtensionDeadline, &outstanding, &notes,
		&status, &resolvedGrade, &row.ResolvedAt, &row.ResolvedBy,
		&row.Reminder30dSentAt, &row.Reminder7dSentAt, &row.Reminder1dSentAt, &row.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	row.OutstandingItemIDs = outstanding
	row.Notes = notes
	row.Status = Status(status)
	row.ResolvedGrade = resolvedGrade
	return &row, nil
}

// Resolve marks an open record resolved with a final grade.
func Resolve(ctx context.Context, pool *pgxpool.Pool, recordID, actorID uuid.UUID, grade string) (*Record, error) {
	now := time.Now().UTC()
	var row Record
	var status string
	var notes, resolvedGrade *string
	var outstanding []uuid.UUID
	err := pool.QueryRow(ctx, `
UPDATE course.incomplete_grade_records
SET status = 'resolved', resolved_grade = $3, resolved_at = $4, resolved_by = $2
WHERE id = $1 AND status = 'open'
RETURNING id, enrollment_id, granted_by, extension_deadline, outstanding_item_ids, notes,
          status, resolved_grade, resolved_at, resolved_by,
          reminder_30d_sent_at, reminder_7d_sent_at, reminder_1d_sent_at, created_at
`, recordID, actorID, grade, now).Scan(
		&row.ID, &row.EnrollmentID, &row.GrantedBy, &row.ExtensionDeadline, &outstanding, &notes,
		&status, &resolvedGrade, &row.ResolvedAt, &row.ResolvedBy,
		&row.Reminder30dSentAt, &row.Reminder7dSentAt, &row.Reminder1dSentAt, &row.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	row.OutstandingItemIDs = outstanding
	row.Notes = notes
	row.Status = Status(status)
	row.ResolvedGrade = resolvedGrade
	return &row, nil
}

// ListReport returns incomplete records for the admin/registrar report.
func ListReport(ctx context.Context, pool *pgxpool.Pool, termID *uuid.UUID, status Status) ([]ReportRow, error) {
	q := `
SELECT igr.id, igr.enrollment_id, igr.granted_by, igr.extension_deadline, igr.outstanding_item_ids,
       igr.notes, igr.status, igr.resolved_grade, igr.resolved_at, igr.resolved_by,
       igr.reminder_30d_sent_at, igr.reminder_7d_sent_at, igr.reminder_1d_sent_at, igr.created_at,
       ce.user_id,
       COALESCE(NULLIF(TRIM(u.display_name), ''), u.email) AS student_name,
       c.id, c.course_code, COALESCE(c.title, c.course_code) AS course_title,
       c.term_id
FROM course.incomplete_grade_records igr
INNER JOIN course.course_enrollments ce ON ce.id = igr.enrollment_id
INNER JOIN course.courses c ON c.id = ce.course_id
INNER JOIN "user".users u ON u.id = ce.user_id
WHERE igr.status = $1
`
	args := []any{string(status)}
	if termID != nil {
		q += ` AND c.term_id = $2`
		args = append(args, *termID)
	}
	q += ` ORDER BY igr.extension_deadline ASC, student_name ASC`

	rows, err := pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []ReportRow
	for rows.Next() {
		var rr ReportRow
		var statusStr string
		var notes, resolvedGrade *string
		var outstanding []uuid.UUID
		if err := rows.Scan(
			&rr.ID, &rr.EnrollmentID, &rr.GrantedBy, &rr.ExtensionDeadline, &outstanding, &notes,
			&statusStr, &resolvedGrade, &rr.ResolvedAt, &rr.ResolvedBy,
			&rr.Reminder30dSentAt, &rr.Reminder7dSentAt, &rr.Reminder1dSentAt, &rr.CreatedAt,
			&rr.StudentUserID, &rr.StudentName, &rr.CourseID, &rr.CourseCode, &rr.CourseTitle, &rr.TermID,
		); err != nil {
			return nil, err
		}
		rr.OutstandingItemIDs = outstanding
		rr.Notes = notes
		rr.Status = Status(statusStr)
		rr.ResolvedGrade = resolvedGrade
		out = append(out, rr)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	for i := range out {
		instIDs, err := listCourseInstructorIDs(ctx, pool, out[i].CourseID)
		if err != nil {
			return nil, err
		}
		out[i].InstructorIDs = instIDs
		titles, err := listItemTitles(ctx, pool, out[i].OutstandingItemIDs)
		if err != nil {
			return nil, err
		}
		out[i].OutstandingTitles = titles
	}
	return out, nil
}

func listCourseInstructorIDs(ctx context.Context, pool *pgxpool.Pool, courseID uuid.UUID) ([]uuid.UUID, error) {
	rows, err := pool.Query(ctx, `
SELECT DISTINCT ce.user_id
FROM course.course_enrollments ce
WHERE ce.course_id = $1 AND ce.active AND ce.role IN ('teacher', 'instructor', 'owner')
`, courseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func listItemTitles(ctx context.Context, pool *pgxpool.Pool, itemIDs []uuid.UUID) ([]string, error) {
	if len(itemIDs) == 0 {
		return nil, nil
	}
	rows, err := pool.Query(ctx, `
SELECT title FROM course.course_structure_items WHERE id = ANY($1::uuid[]) ORDER BY title ASC
`, itemIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var titles []string
	for rows.Next() {
		var t string
		if err := rows.Scan(&t); err != nil {
			return nil, err
		}
		titles = append(titles, t)
	}
	return titles, rows.Err()
}

// ListDueReminders returns open records needing 30/7/1-day reminders as of today.
func ListDueReminders(ctx context.Context, pool *pgxpool.Pool, today time.Time) ([]ReminderCandidate, error) {
	today = time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, time.UTC)
	rows, err := pool.Query(ctx, `
SELECT igr.id, igr.enrollment_id, igr.granted_by, igr.extension_deadline, igr.outstanding_item_ids,
       igr.notes, igr.status, igr.resolved_grade, igr.resolved_at, igr.resolved_by,
       igr.reminder_30d_sent_at, igr.reminder_7d_sent_at, igr.reminder_1d_sent_at, igr.created_at,
       ce.user_id,
       COALESCE(NULLIF(TRIM(u.display_name), ''), u.email) AS student_name,
       c.course_code, COALESCE(c.title, c.course_code) AS course_title, c.id
FROM course.incomplete_grade_records igr
INNER JOIN course.course_enrollments ce ON ce.id = igr.enrollment_id
INNER JOIN course.courses c ON c.id = ce.course_id
INNER JOIN "user".users u ON u.id = ce.user_id
WHERE igr.status = 'open'
`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []ReminderCandidate
	for rows.Next() {
		var rc ReminderCandidate
		var statusStr string
		var notes, resolvedGrade *string
		var outstanding []uuid.UUID
		var courseID uuid.UUID
		if err := rows.Scan(
			&rc.ID, &rc.EnrollmentID, &rc.GrantedBy, &rc.ExtensionDeadline, &outstanding, &notes,
			&statusStr, &resolvedGrade, &rc.ResolvedAt, &rc.ResolvedBy,
			&rc.Reminder30dSentAt, &rc.Reminder7dSentAt, &rc.Reminder1dSentAt, &rc.CreatedAt,
			&rc.StudentUserID, &rc.StudentName, &rc.CourseCode, &rc.CourseTitle, &courseID,
		); err != nil {
			return nil, err
		}
		rc.OutstandingItemIDs = outstanding
		rc.Notes = notes
		rc.Status = Status(statusStr)
		rc.ResolvedGrade = resolvedGrade

		deadline := time.Date(rc.ExtensionDeadline.Year(), rc.ExtensionDeadline.Month(), rc.ExtensionDeadline.Day(), 0, 0, 0, 0, time.UTC)
		days := int(deadline.Sub(today).Hours() / 24)

		var kind string
		switch {
		case days <= 30 && days > 7 && rc.Reminder30dSentAt == nil:
			kind = "30d"
		case days <= 7 && days > 1 && rc.Reminder7dSentAt == nil:
			kind = "7d"
		case days <= 1 && days >= 0 && rc.Reminder1dSentAt == nil:
			kind = "1d"
		default:
			continue
		}
		rc.DaysRemaining = days
		rc.ReminderKind = kind
		instIDs, err := listCourseInstructorIDs(ctx, pool, courseID)
		if err != nil {
			return nil, err
		}
		rc.InstructorIDs = instIDs
		out = append(out, rc)
	}
	return out, rows.Err()
}

// MarkReminderSent records that a reminder was dispatched.
func MarkReminderSent(ctx context.Context, pool *pgxpool.Pool, recordID uuid.UUID, kind string, sentAt time.Time) error {
	col := ""
	switch kind {
	case "30d":
		col = "reminder_30d_sent_at"
	case "7d":
		col = "reminder_7d_sent_at"
	case "1d":
		col = "reminder_1d_sent_at"
	default:
		return fmt.Errorf("unknown reminder kind %q", kind)
	}
	_, err := pool.Exec(ctx, fmt.Sprintf(`
UPDATE course.incomplete_grade_records SET %s = $2 WHERE id = $1
`, col), recordID, sentAt)
	return err
}

// LapseOverdue marks open records past their extension deadline as lapsed.
func LapseOverdue(ctx context.Context, pool *pgxpool.Pool, today time.Time) (int64, error) {
	today = time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, time.UTC)
	tag, err := pool.Exec(ctx, `
UPDATE course.incomplete_grade_records
SET status = 'lapsed'
WHERE status = 'open' AND extension_deadline < $1
`, today)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}
