package homeschoolrecords

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lextures/lextures/server/internal/repos/studentprogress"
)

// HoursEntry is a parent-entered offline/manual hours row.
type HoursEntry struct {
	ID            uuid.UUID
	ParentUserID  uuid.UUID
	StudentUserID uuid.UUID
	EntryDate     time.Time // date at midnight UTC
	Subject       string
	CourseID      *uuid.UUID
	Hours         float64
	Notes         string
	Source        string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// AutoHoursRow is instructional time derived from learning activity for one day+subject.
type AutoHoursRow struct {
	EntryDate time.Time
	Subject   string
	CourseID  *uuid.UUID
	Hours     float64
}

// CreateHoursInput creates a manual/offline hours entry.
type CreateHoursInput struct {
	ParentUserID  uuid.UUID
	StudentUserID uuid.UUID
	EntryDate     time.Time
	Subject       string
	CourseID      *uuid.UUID
	Hours         float64
	Notes         string
	Source        string
}

// UpdateHoursInput updates a manual/offline hours entry.
type UpdateHoursInput struct {
	EntryDate *time.Time
	Subject   *string
	CourseID  **uuid.UUID // nil = leave; non-nil pointer to nil = clear; pointer to uuid = set
	Hours     *float64
	Notes     *string
}

func normalizeSource(s string) string {
	s = strings.TrimSpace(strings.ToLower(s))
	if s == "offline" {
		return "offline"
	}
	return "manual"
}

func normalizeSubject(s string) string {
	s = strings.TrimSpace(s)
	if len(s) > 200 {
		s = s[:200]
	}
	return s
}

func normalizeNotes(s string) string {
	s = strings.TrimSpace(s)
	if len(s) > 2000 {
		s = s[:2000]
	}
	return s
}

func dateOnlyUTC(t time.Time) time.Time {
	y, m, d := t.UTC().Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

// CreateHours inserts a parent-entered hours row.
func CreateHours(ctx context.Context, pool *pgxpool.Pool, in CreateHoursInput) (*HoursEntry, error) {
	if in.Hours < 0 || in.Hours > 24 {
		return nil, fmt.Errorf("hours must be between 0 and 24")
	}
	day := dateOnlyUTC(in.EntryDate)
	src := normalizeSource(in.Source)
	subject := normalizeSubject(in.Subject)
	notes := normalizeNotes(in.Notes)
	var e HoursEntry
	err := pool.QueryRow(ctx, `
INSERT INTO "user".homeschool_hours_entries (
    parent_user_id, student_user_id, entry_date, subject, course_id, hours, notes, source
) VALUES ($1, $2, $3::date, $4, $5, $6, $7, $8)
RETURNING id, parent_user_id, student_user_id, entry_date, subject, course_id, hours::float8, notes, source, created_at, updated_at
`, in.ParentUserID, in.StudentUserID, day, subject, in.CourseID, in.Hours, notes, src).Scan(
		&e.ID, &e.ParentUserID, &e.StudentUserID, &e.EntryDate, &e.Subject, &e.CourseID, &e.Hours, &e.Notes, &e.Source, &e.CreatedAt, &e.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &e, nil
}

// GetHours returns one entry by id, or nil when missing.
func GetHours(ctx context.Context, pool *pgxpool.Pool, id uuid.UUID) (*HoursEntry, error) {
	var e HoursEntry
	err := pool.QueryRow(ctx, `
SELECT id, parent_user_id, student_user_id, entry_date, subject, course_id, hours::float8, notes, source, created_at, updated_at
FROM "user".homeschool_hours_entries WHERE id = $1
`, id).Scan(
		&e.ID, &e.ParentUserID, &e.StudentUserID, &e.EntryDate, &e.Subject, &e.CourseID, &e.Hours, &e.Notes, &e.Source, &e.CreatedAt, &e.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &e, nil
}

// UpdateHours updates fields on a manual entry owned by the parent for the student.
func UpdateHours(ctx context.Context, pool *pgxpool.Pool, parentID, studentID, id uuid.UUID, in UpdateHoursInput) (*HoursEntry, error) {
	cur, err := GetHours(ctx, pool, id)
	if err != nil {
		return nil, err
	}
	if cur == nil || cur.ParentUserID != parentID || cur.StudentUserID != studentID {
		return nil, nil
	}
	day := cur.EntryDate
	if in.EntryDate != nil {
		day = dateOnlyUTC(*in.EntryDate)
	}
	subject := cur.Subject
	if in.Subject != nil {
		subject = normalizeSubject(*in.Subject)
	}
	courseID := cur.CourseID
	if in.CourseID != nil {
		courseID = *in.CourseID
	}
	hours := cur.Hours
	if in.Hours != nil {
		if *in.Hours < 0 || *in.Hours > 24 {
			return nil, fmt.Errorf("hours must be between 0 and 24")
		}
		hours = *in.Hours
	}
	notes := cur.Notes
	if in.Notes != nil {
		notes = normalizeNotes(*in.Notes)
	}
	var e HoursEntry
	err = pool.QueryRow(ctx, `
UPDATE "user".homeschool_hours_entries SET
    entry_date = $1::date, subject = $2, course_id = $3, hours = $4, notes = $5, updated_at = now()
WHERE id = $6 AND parent_user_id = $7 AND student_user_id = $8
RETURNING id, parent_user_id, student_user_id, entry_date, subject, course_id, hours::float8, notes, source, created_at, updated_at
`, day, subject, courseID, hours, notes, id, parentID, studentID).Scan(
		&e.ID, &e.ParentUserID, &e.StudentUserID, &e.EntryDate, &e.Subject, &e.CourseID, &e.Hours, &e.Notes, &e.Source, &e.CreatedAt, &e.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &e, nil
}

// DeleteHours removes a manual entry owned by the parent for the student.
func DeleteHours(ctx context.Context, pool *pgxpool.Pool, parentID, studentID, id uuid.UUID) (bool, error) {
	tag, err := pool.Exec(ctx, `
DELETE FROM "user".homeschool_hours_entries
WHERE id = $1 AND parent_user_id = $2 AND student_user_id = $3
`, id, parentID, studentID)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}

// ListManualHours returns parent-entered entries in [from, to] inclusive by calendar date.
func ListManualHours(ctx context.Context, pool *pgxpool.Pool, studentID uuid.UUID, from, to time.Time) ([]HoursEntry, error) {
	fromD, toD := dateOnlyUTC(from), dateOnlyUTC(to)
	rows, err := pool.Query(ctx, `
SELECT id, parent_user_id, student_user_id, entry_date, subject, course_id, hours::float8, notes, source, created_at, updated_at
FROM "user".homeschool_hours_entries
WHERE student_user_id = $1 AND entry_date >= $2::date AND entry_date <= $3::date
ORDER BY entry_date DESC, subject ASC, created_at DESC
`, studentID, fromD, toD)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]HoursEntry, 0)
	for rows.Next() {
		var e HoursEntry
		if err := rows.Scan(&e.ID, &e.ParentUserID, &e.StudentUserID, &e.EntryDate, &e.Subject, &e.CourseID, &e.Hours, &e.Notes, &e.Source, &e.CreatedAt, &e.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// ListAutoHours aggregates engagement activity into hours by day and course/subject.
// Heartbeats count as studentprogress.HeartbeatSeconds; time_on_task uses the stored seconds.
func ListAutoHours(ctx context.Context, pool *pgxpool.Pool, studentID uuid.UUID, from, to time.Time) ([]AutoHoursRow, error) {
	fromT := dateOnlyUTC(from)
	toT := dateOnlyUTC(to).Add(24 * time.Hour) // exclusive end
	rows, err := pool.Query(ctx, `
SELECT (e.occurred_at AT TIME ZONE 'UTC')::date AS day,
       COALESCE(c.title, 'General') AS subject,
       e.course_id,
       (SUM(CASE
           WHEN e.event_type = 'time_on_task' THEN GREATEST(COALESCE(e.value, 0), 0)
           WHEN e.event_type = 'heartbeat' THEN $4::float8
           ELSE 0 END) / 3600.0)::float8 AS hours
FROM analytics.engagement_events e
LEFT JOIN course.courses c ON c.id = e.course_id
WHERE e.user_id = $1
  AND e.occurred_at >= $2 AND e.occurred_at < $3
  AND e.event_type IN ('time_on_task', 'heartbeat')
GROUP BY 1, 2, 3
HAVING SUM(CASE
           WHEN e.event_type = 'time_on_task' THEN GREATEST(COALESCE(e.value, 0), 0)
           WHEN e.event_type = 'heartbeat' THEN $4::float8
           ELSE 0 END) > 0
ORDER BY 1 DESC, 2 ASC
`, studentID, fromT, toT, float64(studentprogress.HeartbeatSeconds))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]AutoHoursRow, 0)
	for rows.Next() {
		var r AutoHoursRow
		if err := rows.Scan(&r.EntryDate, &r.Subject, &r.CourseID, &r.Hours); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}
