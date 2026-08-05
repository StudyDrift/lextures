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
//
// When the course timezone is LOCAL, due_at is treated as a floating wall clock
// stored in UTC components, and lateness is evaluated in each submitter's
// profile timezone (falling back to UTC if unset).
func MarkOverdueLate(ctx context.Context, pool *pgxpool.Pool, now time.Time) (int64, error) {
	tag, err := pool.Exec(ctx, `
UPDATE course.module_assignment_submissions s
SET is_late = true
FROM course.course_structure_items i
JOIN course.courses c ON c.id = i.course_id
JOIN "user".users u ON u.id = s.submitted_by
WHERE i.id = s.module_item_id
  AND i.due_at IS NOT NULL
  AND (
    CASE
      WHEN UPPER(TRIM(COALESCE(c.course_timezone, ''))) = 'LOCAL' THEN
        ((i.due_at AT TIME ZONE 'UTC') AT TIME ZONE COALESCE(NULLIF(TRIM(u.timezone), ''), 'UTC'))
      ELSE i.due_at
    END
  ) < $1
  AND s.submitted_at > (
    CASE
      WHEN UPPER(TRIM(COALESCE(c.course_timezone, ''))) = 'LOCAL' THEN
        ((i.due_at AT TIME ZONE 'UTC') AT TIME ZONE COALESCE(NULLIF(TRIM(u.timezone), ''), 'UTC'))
      ELSE i.due_at
    END
  )
  AND s.is_late = false
`, now)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}
