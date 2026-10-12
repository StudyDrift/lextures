package studentprogress

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// HeartbeatSeconds is the time credited per engagement heartbeat event (matches the
// learner-profile content-modality deriver).
const HeartbeatSeconds = 30

// WorkItemRow is one published, non-archived assignment or quiz with the learner's
// first completion time (assignment submission or submitted quiz attempt).
type WorkItemRow struct {
	ItemID      uuid.UUID
	Title       string
	Kind        string
	DueAt       *time.Time
	CompletedAt *time.Time
}

// ListWorkItems returns every published, non-archived assignment and quiz in the course
// with the learner's completion time (nil when not yet completed).
func ListWorkItems(ctx context.Context, pool *pgxpool.Pool, courseID, userID uuid.UUID) ([]WorkItemRow, error) {
	rows, err := pool.Query(ctx, `
SELECT csi.id, csi.title, csi.kind, csi.due_at,
       CASE
           WHEN csi.kind = 'assignment' THEN (
               SELECT MIN(mas.submitted_at) FROM course.module_assignment_submissions mas
               WHERE mas.module_item_id = csi.id AND mas.submitted_by = $2
           )
           ELSE (
               SELECT MIN(qa.submitted_at) FROM course.quiz_attempts qa
               WHERE qa.structure_item_id = csi.id AND qa.student_user_id = $2 AND qa.status = 'submitted'
           )
       END AS completed_at
FROM course.course_structure_items csi
WHERE csi.course_id = $1
  AND csi.published AND NOT csi.archived
  AND csi.kind IN ('assignment', 'quiz')
ORDER BY csi.due_at NULLS LAST, csi.sort_order ASC
`, courseID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]WorkItemRow, 0)
	for rows.Next() {
		var r WorkItemRow
		if err := rows.Scan(&r.ItemID, &r.Title, &r.Kind, &r.DueAt, &r.CompletedAt); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// TimeSpentSeconds sums the learner's engagement time in [from, to): explicit
// time_on_task values plus HeartbeatSeconds per heartbeat event.
func TimeSpentSeconds(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID, from, to time.Time) (int, error) {
	var secs float64
	err := pool.QueryRow(ctx, `
SELECT COALESCE(SUM(CASE
           WHEN event_type = 'time_on_task' THEN GREATEST(COALESCE(value, 0), 0)
           WHEN event_type = 'heartbeat' THEN $4::float8
           ELSE 0 END), 0)::float8
FROM analytics.engagement_events
WHERE user_id = $1
  AND occurred_at >= $2 AND occurred_at < $3
  AND event_type IN ('time_on_task', 'heartbeat')
`, userID, from, to, float64(HeartbeatSeconds)).Scan(&secs)
	if err != nil {
		return 0, err
	}
	return int(secs), nil
}
