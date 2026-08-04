package coursechecklist

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ListDismissNotesForUser returns dismiss notes authored by the user (DSAR export).
func ListDismissNotesForUser(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID) ([]DismissNoteExport, error) {
	rows, err := pool.Query(ctx, `
SELECT s.course_id, c.course_code, s.item_id, s.dismiss_reason, s.dismiss_note, s.dismissed_at
FROM course.course_checklist_item_state s
JOIN course.courses c ON c.id = s.course_id
WHERE s.dismissed_by_user_id = $1
  AND s.dismissed_at IS NOT NULL
  AND s.dismiss_note <> ''
ORDER BY s.dismissed_at DESC
`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []DismissNoteExport
	for rows.Next() {
		var r DismissNoteExport
		if err := rows.Scan(&r.CourseID, &r.CourseCode, &r.ItemID, &r.Reason, &r.Note, &r.DismissedAt); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}
