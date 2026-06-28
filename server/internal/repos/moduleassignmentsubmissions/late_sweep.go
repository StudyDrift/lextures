package moduleassignmentsubmissions

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// MarkOverdueLate sets is_late = true on submissions that were turned in after
// the assignment due date, returning the number newly marked. The due date lives
// on course.course_structure_items.due_at; only items with a due date in the
// past are considered. Idempotent: already-late rows are skipped, so the sweep
// is safe to re-run (plan 17.4 FR-4, AC-1).
func MarkOverdueLate(ctx context.Context, pool *pgxpool.Pool, now time.Time) (int64, error) {
	tag, err := pool.Exec(ctx, `
UPDATE course.module_assignment_submissions s
SET is_late = true
FROM course.course_structure_items i
WHERE i.id = s.module_item_id
  AND i.due_at IS NOT NULL
  AND i.due_at < $1
  AND s.submitted_at > i.due_at
  AND s.is_late = false
`, now)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}
