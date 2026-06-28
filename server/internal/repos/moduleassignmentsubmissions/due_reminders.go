package moduleassignmentsubmissions

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// DueReminder is one assignment-due reminder target: a student enrolled in a
// course with an assignment due soon and no submission yet (plan 17.4 FR-4
// due_date_reminder).
type DueReminder struct {
	StudentUserID   uuid.UUID
	CourseID        uuid.UUID
	ModuleItemID    uuid.UUID
	CourseTitle     string
	AssignmentTitle string
	DueAt           time.Time
}

// ListUpcomingDueReminders returns reminder targets for assignments whose due
// date falls in (now, now+window]. Only enrolled students without a submission
// for the item are included, so a student who already turned the work in is not
// nagged. The set is bounded by limit to keep one sweep's enqueue work small.
func ListUpcomingDueReminders(ctx context.Context, pool *pgxpool.Pool, now time.Time, window time.Duration, limit int) ([]DueReminder, error) {
	if limit <= 0 || limit > 5000 {
		limit = 1000
	}
	until := now.Add(window)
	rows, err := pool.Query(ctx, `
SELECT e.user_id, c.id, i.id, c.title, i.title, i.due_at
FROM course.course_structure_items i
JOIN course.courses c ON c.id = i.course_id
JOIN course.course_enrollments e ON e.course_id = c.id AND e.role = 'student'
WHERE i.due_at IS NOT NULL
  AND i.due_at > $1
  AND i.due_at <= $2
  AND NOT EXISTS (
      SELECT 1 FROM course.module_assignment_submissions s
      WHERE s.module_item_id = i.id AND s.submitted_by = e.user_id
  )
ORDER BY i.due_at
LIMIT $3
`, now, until, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []DueReminder
	for rows.Next() {
		var d DueReminder
		if err := rows.Scan(&d.StudentUserID, &d.CourseID, &d.ModuleItemID, &d.CourseTitle, &d.AssignmentTitle, &d.DueAt); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}
