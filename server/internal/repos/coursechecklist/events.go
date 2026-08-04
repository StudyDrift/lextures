package coursechecklist

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func insertEventTx(ctx context.Context, tx pgx.Tx, courseID uuid.UUID, itemID, action string, actor *uuid.UUID, reason string, at time.Time) error {
	_, err := tx.Exec(ctx, `
INSERT INTO course.course_checklist_events (course_id, item_id, action, actor_user_id, reason, occurred_at)
VALUES ($1, $2, $3, $4, $5, $6)
`, courseID, itemID, action, actor, reason, at)
	return err
}

// ListEvents returns the most recent checklist events for a course (newest first).
func ListEvents(ctx context.Context, pool *pgxpool.Pool, courseID uuid.UUID, limit int) ([]Event, error) {
	if limit <= 0 || limit > 100 {
		limit = 100
	}
	rows, err := pool.Query(ctx, `
SELECT id, course_id, item_id, action, actor_user_id, reason, occurred_at
FROM course.course_checklist_events
WHERE course_id = $1
ORDER BY occurred_at DESC
LIMIT $2
`, courseID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Event
	for rows.Next() {
		var e Event
		if err := rows.Scan(&e.ID, &e.CourseID, &e.ItemID, &e.Action, &e.ActorUserID, &e.Reason, &e.OccurredAt); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// DeleteEventsOlderThan removes audit rows older than cutoff (retention sweeper).
func DeleteEventsOlderThan(ctx context.Context, pool *pgxpool.Pool, cutoff time.Time) (int64, error) {
	tag, err := pool.Exec(ctx, `
DELETE FROM course.course_checklist_events WHERE occurred_at < $1
`, cutoff)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}
