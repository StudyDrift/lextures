// Package parentprogress loads the rows for the weekly parent progress email.
package parentprogress

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Parent is one adult who can receive the weekly email.
type Parent struct {
	ID          uuid.UUID
	Email       string
	DisplayName string
	Timezone    *string
}

// Learner is one linked student.
type Learner struct {
	ID   uuid.UUID
	Name string
}

// Item is a course item title pair.
type Item struct {
	Course string
	Title  string
	When   time.Time
}

// GradeChange is one posted or edited score in the window.
type GradeChange struct {
	Course   string
	Title    string
	Previous string
	Next     string
}

// Comment is one visible feedback line.
type Comment struct {
	Course    string
	Title     string
	Body      string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// ListParents returns active parents with at least one active child link.
func ListParents(ctx context.Context, pool *pgxpool.Pool, limit int) ([]Parent, error) {
	if limit <= 0 {
		limit = 2000
	}
	rows, err := pool.Query(ctx, `
SELECT DISTINCT u.id, u.email, COALESCE(NULLIF(TRIM(u.display_name), ''), ''), u.timezone
FROM "user".parent_student_links l
JOIN "user".users u ON u.id = l.parent_user_id
WHERE l.status = 'active'
  AND u.deactivated_at IS NULL
ORDER BY u.id
LIMIT $1
`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Parent
	for rows.Next() {
		var p Parent
		var tz sql.NullString
		if err := rows.Scan(&p.ID, &p.Email, &p.DisplayName, &tz); err != nil {
			return nil, err
		}
		if tz.Valid && tz.String != "" {
			p.Timezone = &tz.String
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// ListLearners returns active linked students for a parent.
func ListLearners(ctx context.Context, pool *pgxpool.Pool, parentID uuid.UUID) ([]Learner, error) {
	rows, err := pool.Query(ctx, `
SELECT s.id, COALESCE(NULLIF(TRIM(s.display_name), ''), 'Learner')
FROM "user".parent_student_links l
JOIN "user".users s ON s.id = l.student_user_id
WHERE l.parent_user_id = $1
  AND l.status = 'active'
  AND s.deactivated_at IS NULL
ORDER BY 2, s.id
`, parentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Learner
	for rows.Next() {
		var l Learner
		if err := rows.Scan(&l.ID, &l.Name); err != nil {
			return nil, err
		}
		out = append(out, l)
	}
	return out, rows.Err()
}

// CompletedItems returns work finished in the window.
func CompletedItems(ctx context.Context, pool *pgxpool.Pool, studentID uuid.UUID, start, end time.Time) ([]Item, error) {
	rows, err := pool.Query(ctx, `
SELECT course_title, item_title, done_at FROM (
  SELECT c.title AS course_title, i.title AS item_title, p.met_at AS done_at
  FROM course.student_item_progress p
  JOIN course.course_enrollments e ON e.id = p.enrollment_id
  JOIN course.enrollment_roles er ON er.role_key = e.role AND er.is_student_equivalent = true
  JOIN course.courses c ON c.id = e.course_id
  JOIN course.course_structure_items i ON i.id = p.item_id
  WHERE e.user_id = $1 AND e.active AND p.status = 'complete'
    AND p.met_at >= $2 AND p.met_at < $3
  UNION
  SELECT c.title, i.title, s.submitted_at
  FROM course.module_assignment_submissions s
  JOIN course.courses c ON c.id = s.course_id
  JOIN course.course_structure_items i ON i.id = s.module_item_id
  WHERE s.submitted_by = $1
    AND s.submitted_at >= $2 AND s.submitted_at < $3
) q
ORDER BY done_at DESC
LIMIT 12
`, studentID, start, end)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanItems(rows)
}

// DueItems returns published work due in [start, end).
func DueItems(ctx context.Context, pool *pgxpool.Pool, studentID uuid.UUID, start, end time.Time) ([]Item, error) {
	rows, err := pool.Query(ctx, `
SELECT c.title, i.title, i.due_at
FROM course.course_enrollments e
JOIN course.enrollment_roles er ON er.role_key = e.role AND er.is_student_equivalent = true
JOIN course.courses c ON c.id = e.course_id
JOIN course.course_structure_items i ON i.course_id = c.id
LEFT JOIN course.student_item_progress p
  ON p.enrollment_id = e.id AND p.item_id = i.id AND p.status = 'complete'
WHERE e.user_id = $1 AND e.active
  AND i.published AND NOT i.archived
  AND i.due_at >= $2 AND i.due_at < $3
  AND p.id IS NULL
ORDER BY i.due_at
LIMIT 12
`, studentID, start, end)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanItems(rows)
}

// GradeChanges returns score edits in the window.
func GradeChanges(ctx context.Context, pool *pgxpool.Pool, studentID uuid.UUID, start, end time.Time) ([]GradeChange, error) {
	rows, err := pool.Query(ctx, `
SELECT c.title, i.title,
       COALESCE(g.previous_score::text, ''),
       COALESCE(g.new_score::text, '')
FROM course.grade_audit_events g
JOIN course.courses c ON c.id = g.course_id
JOIN course.course_structure_items i ON i.id = g.assignment_id
WHERE g.student_id = $1
  AND g.changed_at >= $2 AND g.changed_at < $3
  AND g.action IN ('created', 'updated', 'posted')
  AND g.new_score IS NOT NULL
ORDER BY g.changed_at DESC
LIMIT 12
`, studentID, start, end)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []GradeChange
	for rows.Next() {
		var g GradeChange
		if err := rows.Scan(&g.Course, &g.Title, &g.Previous, &g.Next); err != nil {
			return nil, err
		}
		out = append(out, g)
	}
	return out, rows.Err()
}

// TimeSpentSeconds sums on-task time and parent-entered hours in the window.
func TimeSpentSeconds(ctx context.Context, pool *pgxpool.Pool, studentID uuid.UUID, start, end time.Time) (int, error) {
	var task sql.NullInt64
	err := pool.QueryRow(ctx, `
SELECT COALESCE(SUM(s.total_time_on_task_s), 0)
FROM analytics.engagement_summaries s
JOIN course.course_enrollments e ON e.id = s.enrollment_id
WHERE e.user_id = $1
  AND s.summary_date >= $2::date
  AND s.summary_date < $3::date
`, studentID, start, end).Scan(&task)
	if err != nil {
		return 0, err
	}
	var hours sql.NullFloat64
	err = pool.QueryRow(ctx, `
SELECT COALESCE(SUM(hours), 0)
FROM "user".homeschool_hours_entries
WHERE student_user_id = $1
  AND entry_date >= $2::date
  AND entry_date < $3::date
`, studentID, start, end).Scan(&hours)
	if err != nil {
		return 0, err
	}
	total := 0
	if task.Valid {
		total += int(task.Int64)
	}
	if hours.Valid {
		total += int(hours.Float64 * 3600)
	}
	return total, nil
}

// RecentComments returns posted feedback updated in the window. Callers filter comment timestamps.
func RecentComments(ctx context.Context, pool *pgxpool.Pool, studentID uuid.UUID, start time.Time) ([]Comment, error) {
	rows, err := pool.Query(ctx, `
SELECT c.title, i.title, cg.instructor_comments_json, cg.updated_at
FROM course.course_grades cg
JOIN course.courses c ON c.id = cg.course_id
JOIN course.course_structure_items i ON i.id = cg.module_item_id
WHERE cg.student_user_id = $1
  AND cg.posted_at IS NOT NULL
  AND cg.updated_at >= $2
  AND cg.instructor_comments_json IS NOT NULL
ORDER BY cg.updated_at DESC
LIMIT 40
`, studentID, start)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Comment
	for rows.Next() {
		var c Comment
		var raw []byte
		if err := rows.Scan(&c.Course, &c.Title, &raw, &c.UpdatedAt); err != nil {
			return nil, err
		}
		c.Body = string(raw)
		out = append(out, c)
	}
	return out, rows.Err()
}

// Claim records that this parent was handled for the local send date.
func Claim(ctx context.Context, pool *pgxpool.Pool, parentID uuid.UUID, weekStart time.Time) (bool, error) {
	var id uuid.UUID
	err := pool.QueryRow(ctx, `
INSERT INTO settings.parent_weekly_progress_sends (parent_user_id, week_start)
VALUES ($1, $2::date)
ON CONFLICT (parent_user_id, week_start) DO NOTHING
RETURNING parent_user_id
`, parentID, weekStart).Scan(&id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// Release removes a claim so a failed send can retry.
func Release(ctx context.Context, pool *pgxpool.Pool, parentID uuid.UUID, weekStart time.Time) error {
	_, err := pool.Exec(ctx, `
DELETE FROM settings.parent_weekly_progress_sends
WHERE parent_user_id = $1 AND week_start = $2::date
`, parentID, weekStart)
	return err
}

func scanItems(rows interface {
	Next() bool
	Scan(dest ...any) error
	Err() error
}) ([]Item, error) {
	var out []Item
	for rows.Next() {
		var it Item
		if err := rows.Scan(&it.Course, &it.Title, &it.When); err != nil {
			return nil, err
		}
		out = append(out, it)
	}
	return out, rows.Err()
}
